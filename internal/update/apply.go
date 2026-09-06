// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package update

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// ExtractBinary pulls the application binary out of a downloaded archive.
// It picks the first entry whose base name starts with the slug and is
// marked executable (or is the only regular file).
func ExtractBinary(archivePath, destDir, slug string) (string, error) {
	switch {
	case strings.HasSuffix(archivePath, ".zip"):
		return extractZip(archivePath, destDir, slug)
	case strings.HasSuffix(archivePath, ".tar.gz"), strings.HasSuffix(archivePath, ".tgz"):
		return extractTarGz(archivePath, destDir, slug)
	}
	return "", fmt.Errorf("unsupported archive type: %s", archivePath)
}

func binaryEntry(name, slug string) bool {
	base := filepath.Base(name)
	if base == slug || base == slug+".exe" || base == slug+"-server" || base == slug+"-server.exe" {
		return true
	}
	return strings.HasPrefix(base, slug) && !strings.Contains(base, ".")
}

// stagedName is the fixed output name for an extracted binary. Archive
// entry names are never used in the destination path, so a hostile archive
// cannot redirect the write (zip-slip) no matter what it contains.
func stagedName(destDir, slug string) string {
	if runtime.GOOS == "windows" {
		return filepath.Join(destDir, slug+".exe")
	}
	return filepath.Join(destDir, slug)
}

func extractTarGz(path, destDir, slug string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return "", err
	}
	tr := tar.NewReader(gz)
	sawFile := false
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		sawFile = true
		if binaryEntry(hdr.Name, slug) {
			return writeExtracted(tr, stagedName(destDir, slug), hdr.FileInfo().Mode())
		}
	}
	if !sawFile {
		return "", errors.New("archive contains no files")
	}
	return "", fmt.Errorf("no %s binary found in archive", slug)
}

func extractZip(path, destDir, slug string) (string, error) {
	zr, err := zip.OpenReader(path)
	if err != nil {
		return "", err
	}
	defer zr.Close()
	for _, f := range zr.File {
		if f.FileInfo().IsDir() || !binaryEntry(f.Name, slug) {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return "", err
		}
		out, werr := writeExtracted(rc, stagedName(destDir, slug), f.Mode())
		rc.Close()
		if werr != nil {
			return "", werr
		}
		return out, nil
	}
	return "", fmt.Errorf("no %s binary found in archive", slug)
}

func writeExtracted(r io.Reader, out string, mode os.FileMode) (string, error) {
	if mode == 0 {
		mode = 0o755
	}
	w, err := os.OpenFile(out, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode.Perm()|0o700)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(w, r); err != nil {
		w.Close()
		return "", err
	}
	if err := w.Sync(); err != nil {
		w.Close()
		return "", err
	}
	if err := w.Close(); err != nil {
		return "", err
	}
	return out, nil
}
