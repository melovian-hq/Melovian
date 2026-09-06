// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package democatalog

import (
	"encoding/binary"
	"fmt"
	"hash/fnv"
	"html"
	"io"
	"strings"
)

func CoverSVG(id string, size int) []byte {
	if size <= 0 {
		size = 600
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(id))
	seed := h.Sum32()
	n := uint32(len(palette)) //#nosec G115 -- palette is a tiny fixed table
	c1 := palette[seed%n]
	c2 := palette[(seed/5)%n]
	c3 := palette[(seed/11)%n]
	title, subtitle := Get().CoverLabel(id)
	title = truncateRunes(title, 22)
	subtitle = truncateRunes(subtitle, 28)

	svg := fmt.Sprintf(
		`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 600 600">`+
			`<defs>`+
			`<linearGradient id="bg" x1="0" y1="0" x2="1" y2="1">`+
			`<stop offset="0%%" stop-color="%s"/><stop offset="100%%" stop-color="%s"/>`+
			`</linearGradient>`+
			`<linearGradient id="band" x1="0" y1="0" x2="0" y2="1">`+
			`<stop offset="0%%" stop-color="%s" stop-opacity="0.85"/>`+
			`<stop offset="100%%" stop-color="#0a0a0a" stop-opacity="0.92"/>`+
			`</linearGradient>`+
			`</defs>`+
			`<rect width="600" height="600" fill="url(#bg)"/>`+
			`<circle cx="%d" cy="%d" r="210" fill="%s" fill-opacity="0.22"/>`+
			`<circle cx="%d" cy="%d" r="120" fill="#ffffff" fill-opacity="0.08"/>`+
			`<rect x="0" y="390" width="600" height="210" fill="url(#band)"/>`+
			`<text x="36" y="470" font-family="Georgia, 'Times New Roman', serif" font-size="42" font-weight="700" fill="#fafafa">%s</text>`+
			`<text x="36" y="518" font-family="Helvetica, Arial, sans-serif" font-size="22" fill="#e5e5e5" fill-opacity="0.9">%s</text>`+
			`<rect x="36" y="540" width="72" height="4" fill="#fafafa" fill-opacity="0.55"/>`+
			`</svg>`,
		size, size,
		c1, c2, c3,
		140+(seed%280), 120+((seed/3)%160), c3,
		380+((seed/7)%160), 200+((seed/13)%140),
		html.EscapeString(title),
		html.EscapeString(subtitle),
	)
	return []byte(svg)
}

func truncateRunes(s string, max int) string {
	r := []rune(strings.TrimSpace(s))
	if len(r) <= max {
		return string(r)
	}
	if max <= 1 {
		return string(r[:max])
	}
	return string(r[:max-1]) + "…"
}

var palette = []string{
	"#1a2744", "#6b2d3c", "#1e4d45", "#4a3428", "#2d2a55",
	"#5c2a2a", "#243b55", "#3d4a28", "#4a2850", "#1f4f48",
	"#5a3a22", "#263356", "#3a2450", "#204060", "#553022",
}

const (
	silentSampleRate = 22050
	silentChannels   = 1
	silentBits       = 16
)

// SilentWAV returns a short silent PCM WAV (about 2 seconds).
// Prefer NewSilentWAVReader for duration-accurate demo streams.
func SilentWAV() []byte {
	header := buildSilentWAVHeader(2)
	dataSize := int(binary.LittleEndian.Uint32(header[40:44]))
	buf := make([]byte, 44+dataSize)
	copy(buf, header)
	return buf
}

func buildSilentWAVHeader(seconds int) []byte {
	if seconds < 1 {
		seconds = 1
	}
	if seconds > 600 {
		seconds = 600
	}
	dataSize := silentSampleRate * silentChannels * (silentBits / 8) * seconds
	buf := make([]byte, 44)
	copy(buf[0:], []byte("RIFF"))
	binary.LittleEndian.PutUint32(buf[4:], uint32(36+dataSize))
	copy(buf[8:], []byte("WAVE"))
	copy(buf[12:], []byte("fmt "))
	binary.LittleEndian.PutUint32(buf[16:], 16)
	binary.LittleEndian.PutUint16(buf[20:], 1)
	binary.LittleEndian.PutUint16(buf[22:], uint16(silentChannels))
	binary.LittleEndian.PutUint32(buf[24:], uint32(silentSampleRate))
	byteRate := silentSampleRate * silentChannels * (silentBits / 8)
	binary.LittleEndian.PutUint32(buf[28:], uint32(byteRate))
	binary.LittleEndian.PutUint16(buf[32:], uint16(silentChannels*(silentBits/8)))
	binary.LittleEndian.PutUint16(buf[34:], silentBits)
	copy(buf[36:], []byte("data"))
	binary.LittleEndian.PutUint32(buf[40:], uint32(dataSize))
	return buf
}

// silentWAVReader streams a silent WAV of the given duration without
// allocating the PCM payload. Supports seeking for HTML audio Range requests.
type silentWAVReader struct {
	header   []byte
	dataSize int64
	off      int64
}

// NewSilentWAVReader returns an io.ReadSeeker for seconds of silence.
func NewSilentWAVReader(seconds int) *silentWAVReader {
	header := buildSilentWAVHeader(seconds)
	dataSize := int64(binary.LittleEndian.Uint32(header[40:44]))
	return &silentWAVReader{header: header, dataSize: dataSize}
}

func (r *silentWAVReader) Size() int64 {
	return 44 + r.dataSize
}

func (r *silentWAVReader) Read(p []byte) (int, error) {
	total := r.Size()
	if r.off >= total {
		return 0, io.EOF
	}
	n := 0
	for n < len(p) && r.off < total {
		if r.off < 44 {
			copied := copy(p[n:], r.header[r.off:])
			n += copied
			r.off += int64(copied)
			continue
		}
		// PCM zeros for the rest of the payload.
		remain := total - r.off
		chunk := min(int64(len(p)-n), remain)
		for i := 0; i < int(chunk); i++ {
			p[n+i] = 0
		}
		n += int(chunk)
		r.off += chunk
	}
	if n == 0 {
		return 0, io.EOF
	}
	return n, nil
}

func (r *silentWAVReader) Seek(offset int64, whence int) (int64, error) {
	var abs int64
	switch whence {
	case io.SeekStart:
		abs = offset
	case io.SeekCurrent:
		abs = r.off + offset
	case io.SeekEnd:
		abs = r.Size() + offset
	default:
		return 0, fmt.Errorf("invalid whence")
	}
	if abs < 0 {
		return 0, fmt.Errorf("negative position")
	}
	r.off = abs
	return abs, nil
}
