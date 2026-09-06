// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package metaloader

import (
	"encoding/binary"
	"encoding/hex"
	"hash"
	"io"
	"os"

	"github.com/cespare/xxhash/v2"
)

const contentSampleSize = 256 * 1024

func FileSignature(path string, size int64, mtime int64) string {
	h := xxhash.New()
	_, _ = h.WriteString(path)
	var buf [16]byte
	binary.LittleEndian.PutUint64(buf[0:8], uint64(max(size, 0)))   //#nosec G115 -- negative sizes are clamped before hashing
	binary.LittleEndian.PutUint64(buf[8:16], uint64(max(mtime, 0))) //#nosec G115 -- negative mtimes are clamped before hashing
	_, _ = h.Write(buf[:])
	var sumBuf [16]byte
	return hex.EncodeToString(h.Sum(sumBuf[:0]))
}

func ContentHash(path string, size int64) (string, error) {
	f, err := os.Open(path) //#nosec G304 -- path comes from validated library scan roots
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()

	h := xxhash.New()
	if err := hashFileContent(h, f, size); err != nil {
		return "", err
	}
	var sumBuf [16]byte
	return hex.EncodeToString(h.Sum(sumBuf[:0])), nil
}

func hashFileContent(h hash.Hash, f *os.File, size int64) error {
	if _, err := io.Copy(h, io.LimitReader(f, contentSampleSize)); err != nil {
		return err
	}
	if size > contentSampleSize {
		if _, err := f.Seek(-int64(min(contentSampleSize, size)), io.SeekEnd); err != nil {
			return err
		}
		if _, err := io.Copy(h, io.LimitReader(f, contentSampleSize)); err != nil {
			return err
		}
	}
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], uint64(max(size, 0))) //#nosec G115 -- negative sizes are clamped before hashing
	_, err := h.Write(buf[:])
	return err
}
