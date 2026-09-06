// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package update

import (
	"bytes"
	"compress/bzip2"
	"errors"
	"fmt"
	"io"
)

// bsdiff40 patch applier. Release CI produces .bspatch deltas between the
// raw installed binary of the previous tag and the new one, so a client can
// patch its executable in place instead of downloading a full archive.
//
// Format: 32-byte header ("BSDIFF40", three little-endian-magnitude int64
// lengths for ctrl size, diff size, new size), then three bzip2 streams:
// control triples (add-len, extra-len, seek-adjust), diff bytes, extra
// bytes.

const bsdiffMagic = "BSDIFF40"
const bsdiffHeaderLen = 32

func offtin(buf []byte) int64 {
	if len(buf) < 8 {
		return 0
	}
	y := int64(buf[7] & 0x7F)
	for i := 6; i >= 0; i-- {
		y = y*256 + int64(buf[i])
	}
	if buf[7]&0x80 != 0 {
		y = -y
	}
	return y
}

// ApplyPatch applies a bsdiff40 patch to oldData and returns the new bytes.
func ApplyPatch(oldData, patch []byte) ([]byte, error) {
	if len(patch) < bsdiffHeaderLen || string(patch[:8]) != bsdiffMagic {
		return nil, errors.New("not a BSDIFF40 patch")
	}
	ctrlLen := offtin(patch[8:16])
	diffLen := offtin(patch[16:24])
	newSize := offtin(patch[24:32])
	if ctrlLen < 0 || diffLen < 0 || newSize < 0 {
		return nil, errors.New("corrupt patch header")
	}
	end := int64(len(patch))
	if bsdiffHeaderLen+ctrlLen > end || bsdiffHeaderLen+ctrlLen+diffLen > end {
		return nil, errors.New("truncated patch")
	}
	ctrl := patch[bsdiffHeaderLen : bsdiffHeaderLen+ctrlLen]
	diff := patch[bsdiffHeaderLen+ctrlLen : bsdiffHeaderLen+ctrlLen+diffLen]
	extra := patch[bsdiffHeaderLen+ctrlLen+diffLen:]

	ctrlR, diffR, extraR, err := openPatchStreams(ctrl, diff, extra)
	if err != nil {
		return nil, err
	}

	newData := make([]byte, newSize)
	var oldPos, newPos int64
	for newPos < newSize {
		var triple [24]byte
		if _, err := io.ReadFull(ctrlR, triple[:]); err != nil {
			return nil, fmt.Errorf("read control entry: %w", err)
		}
		addLen := offtin(triple[0:8])
		extraLen := offtin(triple[8:16])
		seekAdj := offtin(triple[16:24])
		if addLen < 0 || extraLen < 0 || newPos+addLen > newSize {
			return nil, errors.New("corrupt control entry")
		}
		if _, err := io.ReadFull(diffR, newData[newPos:newPos+addLen]); err != nil {
			return nil, fmt.Errorf("read diff data: %w", err)
		}
		for i := int64(0); i < addLen; i++ {
			op := oldPos + i
			if op >= 0 && op < int64(len(oldData)) {
				newData[newPos+i] += oldData[op]
			}
		}
		newPos += addLen
		oldPos += addLen
		if newPos+extraLen > newSize {
			return nil, errors.New("corrupt control entry")
		}
		if _, err := io.ReadFull(extraR, newData[newPos:newPos+extraLen]); err != nil {
			return nil, fmt.Errorf("read extra data: %w", err)
		}
		newPos += extraLen
		oldPos += seekAdj
	}
	return newData, nil
}

func openPatchStreams(ctrl, diff, extra []byte) (io.Reader, io.Reader, io.Reader, error) {
	if len(ctrl) == 0 {
		return nil, nil, nil, errors.New("empty control stream")
	}
	var er io.Reader
	if len(extra) > 0 {
		er = bzip2.NewReader(bytes.NewReader(extra))
	} else {
		er = bytes.NewReader(nil)
	}
	return bzip2.NewReader(bytes.NewReader(ctrl)), bzip2.NewReader(bytes.NewReader(diff)), er, nil
}

// offtout writes bsdiff's signed-magnitude little-endian int64. Used by the
// delta generator in CI tooling.
func offtout(x int64, buf []byte) {
	neg := x < 0
	if neg {
		x = -x
	}
	for i := 0; i < 8; i++ {
		buf[i] = byte(x % 256)
		x /= 256
	}
	if neg {
		buf[7] |= 0x80
	}
}

// WritePatchHeader emits a BSDIFF40 header for the given block sizes.
// Exposed for the generator tool. ApplyPatch only reads it.
func WritePatchHeader(w io.Writer, ctrlLen, diffLen, newSize int64) error {
	var hdr [bsdiffHeaderLen]byte
	copy(hdr[:8], bsdiffMagic)
	offtout(ctrlLen, hdr[8:16])
	offtout(diffLen, hdr[16:24])
	offtout(newSize, hdr[24:32])
	_, err := w.Write(hdr[:])
	return err
}
