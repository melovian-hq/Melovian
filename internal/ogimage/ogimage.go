// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

// Package ogimage renders Open Graph share cards (1200x630 PNG) for public
// links. Cards composite a blurred, darkened cover backdrop with a rounded
// artwork tile and Inter text, matching the app's dark theme.
package ogimage

import (
	"bytes"
	_ "embed"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"strings"
	"sync"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"

	xdraw "golang.org/x/image/draw"
)

const (
	Width    = 1200
	Height   = 630
	cardSize = 400
)

// Spec describes one card render.
type Spec struct {
	// Eyebrow is the small tracked label over the title, for example
	// "SHARED PLAYLIST". Empty renders the app wordmark text.
	Eyebrow  string
	Title    string
	Subtitle string
	// Cover is the artwork for the card. Nil renders the branded fallback.
	Cover image.Image
}

//go:embed inter_medium.ttf
var interMedium []byte

var (
	fontOnce sync.Once
	fontErr  error
	// canvasPool reuses the 3MB render target between cards.
	canvasPool = sync.Pool{New: func() any {
		return image.NewRGBA(image.Rect(0, 0, Width, Height))
	}}
)

func loadFont() error {
	fontOnce.Do(func() {
		_, fontErr = opentype.Parse(interMedium)
	})
	return fontErr
}

// face parses a fresh Font per call. Faces that share a Font share its
// glyph raster buffer, so interleaved or concurrent draws corrupt each
// other's glyph masks.
func face(size float64) (font.Face, error) {
	f, err := opentype.Parse(interMedium)
	if err != nil {
		return nil, err
	}
	return opentype.NewFace(f, &opentype.FaceOptions{
		Size:    size,
		DPI:     72,
		Hinting: font.HintingNone,
	})
}

// Colors are premultiplied: color.RGBA requires r,g,b <= a, and image draw
// ops silently misbehave when a source breaks that invariant.
var (
	colWhite     = color.RGBA{255, 255, 255, 255}
	colTitleSub  = color.RGBA{235, 235, 235, 235}
	colEyebrow   = color.RGBA{220, 220, 220, 220}
	colFooter    = color.RGBA{200, 200, 200, 200}
	colAccent    = color.RGBA{0xa4, 0x68, 0xf3, 255}
	colTileBG    = color.RGBA{0x2a, 0x20, 0x3a, 255}
	colTileBG2   = color.RGBA{0x15, 0x11, 0x1e, 255}
	colBackdropA = color.RGBA{10, 9, 14, 140}
	colBackdropB = color.RGBA{8, 7, 12, 216}
)

// Render draws the card and returns PNG bytes.
func Render(spec Spec, wordmark string) ([]byte, error) {
	if err := loadFont(); err != nil {
		return nil, err
	}
	if wordmark == "" {
		wordmark = "Melovian"
	}

	dst := canvasPool.Get().(*image.RGBA)
	defer canvasPool.Put(dst)
	drawBackdrop(dst, spec.Cover)

	const (
		cardX = 88
		cardY = 115
		textX = 560
		textW = Width - textX - 64
	)

	drawShadow(dst, cardX, cardY, cardSize)
	if spec.Cover != nil {
		drawCoverTile(dst, spec.Cover, cardX, cardY)
	} else {
		drawFallbackTile(dst, cardX, cardY)
	}

	eyebrow := strings.ToUpper(strings.TrimSpace(spec.Eyebrow))
	if eyebrow == "" {
		eyebrow = strings.ToUpper(wordmark)
	}
	title := strings.TrimSpace(spec.Title)
	if title == "" {
		title = wordmark
	}
	subtitle := strings.TrimSpace(spec.Subtitle)

	eyebrowFace, _ := face(24)
	titleFace, _ := face(58)
	subFace, _ := face(33)
	footerFace, _ := face(25)
	defer func() {
		closeFace(eyebrowFace)
		closeFace(titleFace)
		closeFace(subFace)
		closeFace(footerFace)
	}()

	titleLines := wrapText(titleFace, title, textW, 2)

	// Center the text block against the card's vertical span.
	blockH := 30 + 34 + len(titleLines)*68
	if subtitle != "" {
		blockH += 18 + 40
	}
	y := cardY + (cardSize-blockH)/2
	drawTracked(dst, eyebrowFace, textX+34, y+30, eyebrow, 4, colEyebrow)
	drawFilledCircle(dst, textX, y+18, 12, colAccent)
	y += 30 + 34
	for _, line := range titleLines {
		y += 68
		drawString(dst, titleFace, textX, y, line, colWhite)
	}
	if subtitle != "" {
		y += 18 + 40
		drawString(dst, subFace, textX, y, fitText(subFace, subtitle, textW), colTitleSub)
	}

	wm := wordmark
	wmWidth := measure(footerFace, wm)
	drawString(dst, footerFace, Width-64-wmWidth, Height-48, wm, colFooter)

	var out bytes.Buffer
	enc := png.Encoder{CompressionLevel: png.BestSpeed}
	if err := enc.Encode(&out, dst); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func closeFace(f font.Face) {
	if f != nil {
		_ = f.Close()
	}
}

// drawBackdrop fills the canvas with the cover scaled to fill, blurred and
// darkened, or a brand gradient when no cover exists.
func drawBackdrop(dst *image.RGBA, cover image.Image) {
	if cover == nil {
		// Brand gradient: deep violet down to near-black.
		top := color.RGBA{0x2b, 0x1d, 0x44, 255}
		bottom := color.RGBA{0x0a, 0x09, 0x0e, 255}
		for y := 0; y < Height; y++ {
			c := lerpColor(top, bottom, float64(y)/(Height-1))
			row := dst.PixOffset(0, y)
			for x := 0; x < Width; x++ {
				i := row + x*4
				dst.Pix[i] = c.R
				dst.Pix[i+1] = c.G
				dst.Pix[i+2] = c.B
				dst.Pix[i+3] = 255
			}
		}
		return
	}

	// Cheap blur: shrink hard, box-blur the small image, stretch back up.
	small := scaleFill(cover, 64, 34)
	boxBlur(small, 2)
	boxBlur(small, 2)
	boxBlur(small, 2)
	xdraw.ApproxBiLinear.Scale(dst, dst.Bounds(), small, small.Bounds(), xdraw.Over, nil)

	// Darkening gradient keeps text legible over bright art.
	for y := 0; y < Height; y++ {
		t := float64(y) / (Height - 1)
		overlay := lerpColor(colBackdropA, colBackdropB, t)
		row := dst.PixOffset(0, y)
		for x := 0; x < Width; x++ {
			i := row + x*4
			a := int(overlay.A)
			dst.Pix[i+0] = toByte((int(dst.Pix[i+0])*(255-a) + int(overlay.R)*a) / 255)
			dst.Pix[i+1] = toByte((int(dst.Pix[i+1])*(255-a) + int(overlay.G)*a) / 255)
			dst.Pix[i+2] = toByte((int(dst.Pix[i+2])*(255-a) + int(overlay.B)*a) / 255)
		}
	}
}

// scaleFill scales src to exactly wxh, center-cropping the longer axis.
func scaleFill(src image.Image, w, h int) *image.RGBA {
	sb := src.Bounds()
	sw, sh := sb.Dx(), sb.Dy()
	if sw <= 0 || sh <= 0 {
		return image.NewRGBA(image.Rect(0, 0, w, h))
	}
	// Crop to target aspect.
	target := float64(w) / float64(h)
	srcAspect := float64(sw) / float64(sh)
	crop := sb
	if srcAspect > target {
		nw := int(float64(sh) * target)
		crop = image.Rect(sb.Min.X+(sw-nw)/2, sb.Min.Y, sb.Min.X+(sw-nw)/2+nw, sb.Max.Y)
	} else if srcAspect < target {
		nh := int(float64(sw) / target)
		crop = image.Rect(sb.Min.X, sb.Min.Y+(sh-nh)/2, sb.Max.X, sb.Min.Y+(sh-nh)/2+nh)
	}
	out := image.NewRGBA(image.Rect(0, 0, w, h))
	xdraw.CatmullRom.Scale(out, out.Bounds(), src, crop, xdraw.Over, nil)
	return out
}

// boxBlur applies one separable box-blur pass in place.
func boxBlur(img *image.RGBA, radius int) {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w < 2*radius+1 || h < 2*radius+1 {
		return
	}
	tmp := image.NewRGBA(b)
	div := 2*radius + 1
	// Horizontal pass.
	for y := 0; y < h; y++ {
		var sr, sg, sb2, sa int
		rowOff := img.PixOffset(0, y)
		for x := -radius; x <= radius; x++ {
			i := rowOff + clamp(x, 0, w-1)*4
			sr += int(img.Pix[i])
			sg += int(img.Pix[i+1])
			sb2 += int(img.Pix[i+2])
			sa += int(img.Pix[i+3])
		}
		for x := 0; x < w; x++ {
			o := tmp.PixOffset(x, y)
			tmp.Pix[o] = toByte(sr / div)
			tmp.Pix[o+1] = toByte(sg / div)
			tmp.Pix[o+2] = toByte(sb2 / div)
			tmp.Pix[o+3] = toByte(sa / div)
			iAdd := rowOff + clamp(x+radius+1, 0, w-1)*4
			iSub := rowOff + clamp(x-radius, 0, w-1)*4
			sr += int(img.Pix[iAdd]) - int(img.Pix[iSub])
			sg += int(img.Pix[iAdd+1]) - int(img.Pix[iSub+1])
			sb2 += int(img.Pix[iAdd+2]) - int(img.Pix[iSub+2])
			sa += int(img.Pix[iAdd+3]) - int(img.Pix[iSub+3])
		}
	}
	// Vertical pass.
	for x := 0; x < w; x++ {
		var sr, sg, sb2, sa int
		for y := -radius; y <= radius; y++ {
			i := tmp.PixOffset(x, clamp(y, 0, h-1))
			sr += int(tmp.Pix[i])
			sg += int(tmp.Pix[i+1])
			sb2 += int(tmp.Pix[i+2])
			sa += int(tmp.Pix[i+3])
		}
		for y := 0; y < h; y++ {
			o := img.PixOffset(x, y)
			img.Pix[o] = toByte(sr / div)
			img.Pix[o+1] = toByte(sg / div)
			img.Pix[o+2] = toByte(sb2 / div)
			img.Pix[o+3] = toByte(sa / div)
			iAdd := tmp.PixOffset(x, clamp(y+radius+1, 0, h-1))
			iSub := tmp.PixOffset(x, clamp(y-radius, 0, h-1))
			sr += int(tmp.Pix[iAdd]) - int(tmp.Pix[iSub])
			sg += int(tmp.Pix[iAdd+1]) - int(tmp.Pix[iSub+1])
			sb2 += int(tmp.Pix[iAdd+2]) - int(tmp.Pix[iSub+2])
			sa += int(tmp.Pix[iAdd+3]) - int(tmp.Pix[iSub+3])
		}
	}
}

// roundedMask builds an alpha mask for a rounded rectangle. Masks for the
// fixed card layout are cached since the pixel loop dominates render cost.
func roundedMask(w, h, r int) *image.Alpha {
	m := image.NewAlpha(image.Rect(0, 0, w, h))
	r2 := float64(r)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			// Distance outside the corner quarter-circle, with 1px of
			// antialiasing so edges do not stair-step.
			dx := math.Max(float64(r)-0.5-float64(x), math.Max(float64(x)-(float64(w-r)-0.5), 0))
			dy := math.Max(float64(r)-0.5-float64(y), math.Max(float64(y)-(float64(h-r)-0.5), 0))
			d := math.Hypot(dx, dy)
			alpha := uint8(0)
			switch {
			case d <= r2-0.5:
				alpha = 255
			case d <= r2+0.5:
				alpha = uint8(255 * (r2 + 0.5 - d))
			}
			m.Pix[y*m.Stride+x] = alpha
		}
	}
	return m
}

var (
	cardMask     = sync.OnceValue(func() *image.Alpha { return roundedMask(cardSize, cardSize, 26) })
	cardEdgeMask = sync.OnceValue(func() *image.Alpha {
		m := roundedMask(cardSize, cardSize, 26)
		subtractMask(m, roundedMask(cardSize-2, cardSize-2, 25))
		return m
	})
	shadowMaskBase = sync.OnceValue(func() *image.Alpha { return roundedMask(cardSize+44, cardSize+44, 26+22) })
)

// drawCoverTile draws the artwork as a rounded card.
func drawCoverTile(dst *image.RGBA, cover image.Image, x, y int) {
	tile := scaleFill(cover, cardSize, cardSize)
	rect := image.Rect(x, y, x+cardSize, y+cardSize)
	draw.DrawMask(dst, rect, tile, image.Point{}, cardMask(), image.Point{}, draw.Over)
	// Hairline edge so dark covers read as a tile.
	edge := image.NewUniform(color.RGBA{26, 26, 26, 26})
	draw.DrawMask(dst, rect, edge, image.Point{}, cardEdgeMask(), image.Point{}, draw.Over)
}

// drawFallbackTile draws the no-artwork tile: brand gradient with a vinyl
// disc motif built from circles.
func drawFallbackTile(dst *image.RGBA, x, y int) {
	tile := image.NewRGBA(image.Rect(0, 0, cardSize, cardSize))
	for ty := 0; ty < cardSize; ty++ {
		c := lerpColor(colTileBG, colTileBG2, float64(ty)/float64(cardSize-1))
		row := tile.PixOffset(0, ty)
		for tx := 0; tx < cardSize; tx++ {
			i := row + tx*4
			tile.Pix[i] = c.R
			tile.Pix[i+1] = c.G
			tile.Pix[i+2] = c.B
			tile.Pix[i+3] = 255
		}
	}
	rect := image.Rect(x, y, x+cardSize, y+cardSize)
	draw.DrawMask(dst, rect, tile, image.Point{}, cardMask(), image.Point{}, draw.Over)

	// Vinyl disc: accent ring, dark groove, accent label.
	cx, cy := x+cardSize/2, y+cardSize/2
	drawFilledCircle(dst, cx, cy, 118, color.RGBA{0x0d, 0x0a, 0x13, 255})
	drawRing(dst, cx, cy, 118, 2, color.RGBA{58, 37, 86, 90})
	drawRing(dst, cx, cy, 84, 1, color.RGBA{14, 14, 14, 14})
	drawRing(dst, cx, cy, 96, 1, color.RGBA{14, 14, 14, 14})
	drawFilledCircle(dst, cx, cy, 42, colAccent)
	drawFilledCircle(dst, cx, cy, 12, colTileBG2)

	edge := image.NewUniform(color.RGBA{20, 20, 20, 20})
	draw.DrawMask(dst, rect, edge, image.Point{}, cardEdgeMask(), image.Point{}, draw.Over)
}

// drawShadow lays a soft dark rounded rect under the card.
func drawShadow(dst *image.RGBA, x, y, size int) {
	mask := shadowMaskBase()
	rect := image.Rect(x-22, y-14, x+size+22, y+size+22)
	draw.DrawMask(dst, rect, image.NewUniform(color.RGBA{0, 0, 0, 44}), image.Point{}, mask, image.Point{}, draw.Over)
}

func subtractMask(outer, inner *image.Alpha) {
	for y := 0; y < inner.Bounds().Dy(); y++ {
		for x := 0; x < inner.Bounds().Dx(); x++ {
			if inner.AlphaAt(x, y).A > 0 {
				outer.SetAlpha(x+1, y+1, color.Alpha{0})
			}
		}
	}
}

// drawRing draws a stroked circle of the given thickness.
func drawRing(dst *image.RGBA, cx, cy, r, thickness int, c color.RGBA) {
	inner := float64(r - thickness)
	outer := float64(r)
	i2, o2 := inner*inner, outer*outer
	for y := -r; y <= r; y++ {
		for x := -r; x <= r; x++ {
			d2 := float64(x*x + y*y)
			if d2 <= o2 && d2 >= i2 {
				px, py := cx+x, cy+y
				if image.Pt(px, py).In(dst.Bounds()) {
					// Premultiplied source-over so thin rings read soft.
					existing := dst.RGBAAt(px, py)
					ia := 255 - int(c.A)
					dst.SetRGBA(px, py, color.RGBA{
						R: toByte(int(c.R) + int(existing.R)*ia/255),
						G: toByte(int(c.G) + int(existing.G)*ia/255),
						B: toByte(int(c.B) + int(existing.B)*ia/255),
						A: 255,
					})
				}
			}
		}
	}
}

func drawFilledCircle(dst *image.RGBA, cx, cy, r int, c color.RGBA) {
	for y := -r; y <= r; y++ {
		for x := -r; x <= r; x++ {
			if x*x+y*y <= r*r {
				px, py := cx+x, cy+y
				if image.Pt(px, py).In(dst.Bounds()) {
					dst.SetRGBA(px, py, c)
				}
			}
		}
	}
}

func drawString(dst *image.RGBA, f font.Face, x, y int, s string, c color.Color) {
	if f == nil {
		return
	}
	d := font.Drawer{
		Dst:  dst,
		Src:  image.NewUniform(c),
		Face: f,
		Dot:  fixed.P(x, y),
	}
	d.DrawString(s)
}

// drawTracked draws uppercase text with letter spacing in pixels.
func drawTracked(dst *image.RGBA, f font.Face, x, y int, s string, tracking int, c color.Color) {
	if f == nil {
		return
	}
	d := font.Drawer{
		Dst:  dst,
		Src:  image.NewUniform(c),
		Face: f,
		Dot:  fixed.P(x, y),
	}
	for _, r := range s {
		d.DrawString(string(r))
		d.Dot.X += fixed.I(tracking)
	}
}

func measure(f font.Face, s string) int {
	if f == nil {
		return 0
	}
	return font.MeasureString(f, s).Ceil()
}

// fitText truncates s with an ellipsis so it fits maxW pixels.
func fitText(f font.Face, s string, maxW int) string {
	if measure(f, s) <= maxW {
		return s
	}
	runes := []rune(s)
	for len(runes) > 0 {
		candidate := string(runes[:]) + "…"
		if measure(f, candidate) <= maxW {
			return candidate
		}
		runes = runes[:len(runes)-1]
	}
	return "…"
}

// wrapText breaks s into at most maxLines lines fitting maxW, adding an
// ellipsis when the text overflows.
func wrapText(f font.Face, s string, maxW, maxLines int) []string {
	words := strings.Fields(s)
	if len(words) == 0 {
		return nil
	}
	var lines []string
	cur := words[0]
	for _, w := range words[1:] {
		if measure(f, cur+" "+w) <= maxW {
			cur += " " + w
			continue
		}
		lines = append(lines, cur)
		cur = w
	}
	lines = append(lines, cur)
	if len(lines) <= maxLines {
		return lines
	}
	lines = lines[:maxLines]
	last := fitText(f, lines[maxLines-1], maxW-8)
	if !strings.HasSuffix(last, "…") {
		last = fitText(f, last+"…", maxW)
	}
	lines[maxLines-1] = last
	return lines
}

func lerpColor(a, b color.RGBA, t float64) color.RGBA {
	return color.RGBA{
		R: uint8(float64(a.R)*(1-t) + float64(b.R)*t),
		G: uint8(float64(a.G)*(1-t) + float64(b.G)*t),
		B: uint8(float64(a.B)*(1-t) + float64(b.B)*t),
		A: uint8(float64(a.A)*(1-t) + float64(b.A)*t),
	}
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// toByte clamps a channel sum into byte range for Pix writes.
func toByte(v int) uint8 {
	return uint8(clamp(v, 0, 255)) //#nosec G115 -- clamped to byte range
}
