// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package transcode

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"melovian/internal/brand"
)

var (
	ErrUnavailable = errors.New("transcoding is not available")
	ffmpegOnce     sync.Once
	ffmpegPath     string
	ffmpegChecked  bool
)

func Available() bool {
	ffmpegOnce.Do(func() {
		path, err := exec.LookPath("ffmpeg")
		if err == nil {
			ffmpegPath = path
			ffmpegChecked = true
		}
	})
	return ffmpegChecked
}

func ShouldTranscode(maxBitRate string, format string) bool {
	if !Available() {
		return false
	}
	if strings.TrimSpace(maxBitRate) == "" {
		return false
	}
	rate, err := strconv.Atoi(maxBitRate)
	return err == nil && rate > 0 && !strings.EqualFold(format, "mp3")
}

func ServeMP3(
	ctx context.Context,
	inputPath string,
	maxBitRate int,
	w http.ResponseWriter,
	r *http.Request,
) error {
	if !Available() {
		return ErrUnavailable
	}
	if maxBitRate <= 0 {
		maxBitRate = 192
	}

	args := []string{
		"-hide_banner",
		"-loglevel", "error",
		"-i", inputPath,
		"-map", "0:a:0",
		"-vn",
		"-ac", "2",
		"-ar", "44100",
		"-b:a", fmt.Sprintf("%dk", maxBitRate),
		"-f", "mp3",
		"pipe:1",
	}

	cmd := exec.CommandContext(ctx, ffmpegPath, args...) //#nosec G204 -- ffmpeg path resolved via LookPath
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}

	w.Header().Set("Content-Type", "audio/mpeg")
	w.Header().Set("Accept-Ranges", "none")
	w.WriteHeader(http.StatusOK)

	_, copyErr := io.Copy(w, stdout)
	waitErr := cmd.Wait()
	if copyErr != nil {
		return copyErr
	}
	if waitErr != nil {
		return waitErr
	}
	return nil
}

func TempOutputPath(inputPath string) string {
	base := strings.TrimSuffix(filepath.Base(inputPath), filepath.Ext(inputPath))
	return filepath.Join(os.TempDir(), brand.Slug+"-transcode-"+base+".mp3")
}
