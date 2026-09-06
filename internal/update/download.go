// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package update

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Download fetches url to w, reporting progress. Returns the number of
// bytes written and the server-reported total (or -1 when unknown).
func Download(ctx context.Context, client *http.Client, url string, w io.Writer, onProgress ProgressFunc) (int64, error) {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Minute}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("User-Agent", userAgent())
	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("download %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("download %s: %s", url, resp.Status)
	}
	total := resp.ContentLength
	var written int64
	buf := make([]byte, 128*1024)
	last := time.Now()
	for {
		n, rerr := resp.Body.Read(buf)
		if n > 0 {
			m, werr := w.Write(buf[:n])
			written += int64(m)
			if werr != nil {
				return written, werr
			}
			if time.Since(last) >= 200*time.Millisecond {
				last = time.Now()
				report(onProgress, Progress{
					Stage:   StageDownload,
					Message: "Downloading update",
					Written: written,
					Total:   total,
				})
			}
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			return written, fmt.Errorf("download %s: %w", url, rerr)
		}
	}
	report(onProgress, Progress{
		Stage:   StageDownload,
		Message: "Download complete",
		Written: written,
		Total:   total,
	})
	return written, nil
}

// DownloadBytes is Download for small in-memory files (checksums, sigs).
func DownloadBytes(ctx context.Context, client *http.Client, url string) ([]byte, error) {
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent())
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download %s: %s", url, resp.Status)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 8<<20))
}
