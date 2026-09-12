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
	ctrl, diff, extra, newSize, err := splitPatch(patch)
	if err != nil {
		return nil, err
	}
	ctrlR, diffR, extraR, err := openPatchStreams(ctrl, diff, extra)
	if err != nil {
		return nil, err
	}

	newData := make([]byte, newSize)
	var oldPos, newPos int64
	for newPos < newSize {
		oldPos, newPos, err = patchStep(oldData, newData, oldPos, newPos, newSize, ctrlR, diffR, extraR)
		if err != nil {
			return nil, err
		}
	}
	return newData, nil
}

// splitPatch validates the BSDIFF40 header and returns the three compressed
// blocks and the expected output size.
func splitPatch(patch []byte) (ctrl, diff, extra []byte, newSize int64, err error) {
	if len(patch) < bsdiffHeaderLen || string(patch[:8]) != bsdiffMagic {
		return nil, nil, nil, 0, errors.New("not a BSDIFF40 patch")
	}
	ctrlLen := offtin(patch[8:16])
	diffLen := offtin(patch[16:24])
	newSize = offtin(patch[24:32])
	if ctrlLen < 0 || diffLen < 0 || newSize < 0 {
		return nil, nil, nil, 0, errors.New("corrupt patch header")
	}
	end := int64(len(patch))
	if bsdiffHeaderLen+ctrlLen > end || bsdiffHeaderLen+ctrlLen+diffLen > end {
		return nil, nil, nil, 0, errors.New("truncated patch")
	}
	return patch[bsdiffHeaderLen : bsdiffHeaderLen+ctrlLen],
		patch[bsdiffHeaderLen+ctrlLen : bsdiffHeaderLen+ctrlLen+diffLen],
		patch[bsdiffHeaderLen+ctrlLen+diffLen:], newSize, nil
}

// patchStep applies one control triple: add addLen diff bytes, copy
// extraLen literal bytes, seek the old cursor by seekAdj.
func patchStep(oldData, newData []byte, oldPos, newPos, newSize int64, ctrlR, diffR, extraR io.Reader) (int64, int64, error) {
	addLen, extraLen, seekAdj, err := readCtrl(ctrlR, newPos, newSize)
	if err != nil {
		return 0, 0, err
	}
	if _, err := io.ReadFull(diffR, newData[newPos:newPos+addLen]); err != nil {
		return 0, 0, fmt.Errorf("read diff data: %w", err)
	}
	for i := range addLen {
		if op := oldPos + i; op >= 0 && op < int64(len(oldData)) {
			newData[newPos+i] += oldData[op]
		}
	}
	if _, err := io.ReadFull(extraR, newData[newPos+addLen:newPos+addLen+extraLen]); err != nil {
		return 0, 0, fmt.Errorf("read extra data: %w", err)
	}
	return oldPos + addLen + seekAdj, newPos + addLen + extraLen, nil
}

// readCtrl reads and validates one 24-byte control triple.
func readCtrl(r io.Reader, newPos, newSize int64) (add, extra, seek int64, err error) {
	var triple [24]byte
	if _, err := io.ReadFull(r, triple[:]); err != nil {
		return 0, 0, 0, fmt.Errorf("read control entry: %w", err)
	}
	add = offtin(triple[0:8])
	extra = offtin(triple[8:16])
	seek = offtin(triple[16:24])
	if add < 0 || extra < 0 || newPos+add > newSize || newPos+add+extra > newSize {
		return 0, 0, 0, errors.New("corrupt control entry")
	}
	return add, extra, seek, nil
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
	for i := range 8 {
		buf[i] = byte(x % 256) //#nosec G115 -- mod 256 is always in byte range
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
