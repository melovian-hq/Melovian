// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package ogimage

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func synthCover() image.Image {
	img := image.NewRGBA(image.Rect(0, 0, 600, 600))
	for y := 0; y < 600; y++ {
		for x := 0; x < 600; x++ {
			t := float64(x+y) / 1200
			img.Set(x, y, color.RGBA{
				R: uint8(180 + 60*t),
				G: uint8(70 + 100*t),
				B: uint8(150 - 80*t),
				A: 255,
			})
		}
	}
	return img
}

func TestRenderWithCover(t *testing.T) {
	out, err := Render(Spec{
		Eyebrow:  "Shared playlist",
		Title:    "Late Night Driving Mix",
		Subtitle: "24 tracks · KEXP Favorites",
		Cover:    synthCover(),
	}, "Melovian")
	if err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(bytes.NewReader(out))
	if err != nil {
		t.Fatalf("rendered bytes are not a png: %v", err)
	}
	if img.Bounds().Dx() != Width || img.Bounds().Dy() != Height {
		t.Fatalf("unexpected size %v", img.Bounds())
	}
}

func TestRenderFallback(t *testing.T) {
	out, err := Render(Spec{
		Eyebrow: "Listen together",
		Title:   "A Very Long Track Title That Should Wrap Onto Two Lines Nicely For The Card",
	}, "Melovian")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := png.Decode(bytes.NewReader(out)); err != nil {
		t.Fatalf("rendered bytes are not a png: %v", err)
	}
}

func TestRenderLongTitleEllipsized(t *testing.T) {
	long := ""
	for i := 0; i < 40; i++ {
		long += "Supercalifragilisticexpialidocious "
	}
	out, err := Render(Spec{Title: long, Cover: synthCover()}, "Melovian")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := png.Decode(bytes.NewReader(out)); err != nil {
		t.Fatalf("rendered bytes are not a png: %v", err)
	}
}

func TestWrapText(t *testing.T) {
	if err := loadFont(); err != nil {
		t.Fatal(err)
	}
	fc, err := face(58)
	if err != nil {
		t.Fatal(err)
	}
	defer closeFace(fc)
	lines := wrapText(fc, "one two three four five six seven eight nine ten", 200, 2)
	if len(lines) > 2 {
		t.Fatalf("expected at most 2 lines, got %d", len(lines))
	}
	if !strings.HasSuffix(lines[len(lines)-1], "…") {
		t.Fatalf("overflow line should end with ellipsis: %q", lines)
	}
}

// TestRenderGolden writes reference cards when OG_GOLDEN_DIR is set so the
// visual output can be inspected by hand.
func TestRenderGolden(t *testing.T) {
	dir := os.Getenv("OG_GOLDEN_DIR")
	if dir == "" {
		t.Skip("OG_GOLDEN_DIR not set")
	}
	for name, spec := range map[string]Spec{
		"cover":   {Eyebrow: "Shared playlist", Title: "Late Night Driving Mix", Subtitle: "24 tracks · KEXP Favorites", Cover: synthCover()},
		"nocover": {Eyebrow: "Listen together", Title: "Midnight Rendezvous", Subtitle: "Some Artist · Some Album"},
	} {
		out, err := Render(spec, "Melovian")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name+".png"), out, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func BenchmarkRenderWithCover(b *testing.B) {
	cover := synthCover()
	spec := Spec{Eyebrow: "Shared playlist", Title: "Late Night Driving Mix", Subtitle: "24 tracks", Cover: cover}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := Render(spec, "Melovian"); err != nil {
			b.Fatal(err)
		}
	}
}
