//go:build ignore

// gen_resize_gif renders an animated GIF that visualises how the keymap
// fills its bounds at different window sizes. It mirrors the math in
// internal/ui/keyboard.go (fitUnit + per-key rect) so the GIF is a
// faithful preview of what the live ebiten/guigui app draws — without
// the GL stack, so it can run headlessly in CI / sandboxes.
//
//	go run tools/gen_resize_gif.go --out docs/resize.gif
package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"log"
	"math"
	"os"
	"strings"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"

	"github.com/rin2yh/keymap-viewer/internal/keymap"
	"github.com/rin2yh/keymap-viewer/internal/viatest"
)

const (
	keyMinUnitPx = 18
	keyGapRatio  = 4.0 / 56.0

	headerH = 28
	pad     = 8
	tabsW   = 60
)

type aabb struct{ x0, y0, x1, y1 float64 }

func keyAABBUnits(k keymap.Key) (x0, y0, x1, y1 float64) {
	w := k.W
	if w <= 0 {
		w = 1
	}
	h := k.H
	if h <= 0 {
		h = 1
	}
	if k.Rotation == 0 {
		return k.X, k.Y, k.X + w, k.Y + h
	}
	cx := k.X + w/2
	cy := k.Y + h/2
	rad := k.Rotation * math.Pi / 180.0
	cosR, sinR := math.Cos(rad), math.Sin(rad)
	dx := cx - k.RotationOriginX
	dy := cy - k.RotationOriginY
	cxR := k.RotationOriginX + dx*cosR - dy*sinR
	cyR := k.RotationOriginY + dx*sinR + dy*cosR
	return cxR - w/2, cyR - h/2, cxR + w/2, cyR + h/2
}

func fitUnit(boundsW, boundsH int, widthU, heightU float64) int {
	if widthU <= 0 || heightU <= 0 || boundsW <= 0 || boundsH <= 0 {
		return 56
	}
	fit := math.Min(float64(boundsW)/widthU, float64(boundsH)/heightU)
	if fit < keyMinUnitPx {
		fit = keyMinUnitPx
	}
	return int(math.Floor(fit))
}

func drawRectBorder(img *image.Paletted, r image.Rectangle, c color.Color) {
	fill := image.NewUniform(c)
	for _, b := range []image.Rectangle{
		image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+1),
		image.Rect(r.Min.X, r.Max.Y-1, r.Max.X, r.Max.Y),
		image.Rect(r.Min.X, r.Min.Y, r.Min.X+1, r.Max.Y),
		image.Rect(r.Max.X-1, r.Min.Y, r.Max.X, r.Max.Y),
	} {
		draw.Draw(img, b.Intersect(img.Bounds()), fill, image.Point{}, draw.Src)
	}
}

func drawText(img *image.Paletted, s string, x, y int, c color.Color) {
	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(c),
		Face: basicfont.Face7x13,
		Dot:  fixed.Point26_6{X: fixed.I(x), Y: fixed.I(y)},
	}
	d.DrawString(s)
}

func renderWindow(def *keymap.Definition, snap *keymap.Snapshot, layer int,
	aabbs []aabb, minX, minY, maxX, maxY float64,
	winW, winH int, palette color.Palette,
) *image.Paletted {
	img := image.NewPaletted(image.Rect(0, 0, winW, winH), palette)
	draw.Draw(img, img.Bounds(), image.NewUniform(palette[0]), image.Point{}, draw.Src)

	headerRect := image.Rect(pad, pad, winW-pad, pad+headerH)
	draw.Draw(img, headerRect, image.NewUniform(palette[3]), image.Point{}, draw.Src)
	drawText(img, fmt.Sprintf("%s  window %d x %d", def.Name, winW, winH), pad+8, pad+headerH/2+5, palette[2])

	bodyMinX := pad
	bodyMinY := pad + headerH + 8
	bodyMaxX := winW - pad
	bodyMaxY := winH - pad

	tabsRect := image.Rect(bodyMinX, bodyMinY, bodyMinX+tabsW, bodyMaxY)
	draw.Draw(img, tabsRect, image.NewUniform(palette[3]), image.Point{}, draw.Src)
	drawText(img, "L0", tabsRect.Min.X+8, tabsRect.Min.Y+18, palette[4])

	kbX0 := bodyMinX + tabsW + 8
	kbY0 := bodyMinY
	kbW := bodyMaxX - kbX0
	kbH := bodyMaxY - kbY0
	if kbW <= 0 || kbH <= 0 {
		return img
	}

	widthU := maxX - minX
	heightU := maxY - minY
	unit := fitUnit(kbW, kbH, widthU, heightU)
	gap := int(math.Round(float64(unit) * keyGapRatio))
	totalW := int(widthU * float64(unit))
	totalH := int(heightU * float64(unit))
	offX := kbX0 + (kbW-totalW)/2 - int(minX*float64(unit))
	offY := kbY0 + (kbH-totalH)/2 - int(minY*float64(unit))

	keyFill := image.NewUniform(palette[1])

	for i, a := range aabbs {
		rect := image.Rect(
			offX+int(a.x0*float64(unit))+gap,
			offY+int(a.y0*float64(unit))+gap,
			offX+int(a.x1*float64(unit))-gap,
			offY+int(a.y1*float64(unit))-gap,
		)
		clipped := rect.Intersect(image.Rect(kbX0, kbY0, bodyMaxX, bodyMaxY))
		if clipped.Empty() {
			continue
		}
		draw.Draw(img, clipped, keyFill, image.Point{}, draw.Src)
		drawRectBorder(img, clipped, palette[2])

		if snap == nil {
			continue
		}
		label := keymap.Label(snap.Keycode(layer, def.Keys[i].Row, def.Keys[i].Col))
		first := label
		if nl := strings.IndexByte(label, '\n'); nl >= 0 {
			first = label[:nl]
		}
		if first == "" {
			continue
		}
		txtW := basicfont.Face7x13.Width * len(first)
		if txtW+4 > rect.Dx() || rect.Dy() < 14 {
			continue
		}
		tx := rect.Min.X + (rect.Dx()-txtW)/2
		ty := rect.Min.Y + (rect.Dy()+10)/2
		drawText(img, first, tx, ty, palette[2])
	}
	return img
}

func main() {
	out := flag.String("out", "docs/resize.gif", "output GIF path")
	flag.Parse()

	def, err := keymap.LoadEmbeddedDefinition()
	if err != nil {
		log.Fatal(err)
	}
	snap := viatest.SampleSnapshot()

	aabbs := make([]aabb, len(def.Keys))
	var minX, minY, maxX, maxY float64
	for i, key := range def.Keys {
		x0, y0, x1, y1 := keyAABBUnits(key)
		aabbs[i] = aabb{x0, y0, x1, y1}
		if i == 0 {
			minX, minY, maxX, maxY = x0, y0, x1, y1
			continue
		}
		if x0 < minX {
			minX = x0
		}
		if y0 < minY {
			minY = y0
		}
		if x1 > maxX {
			maxX = x1
		}
		if y1 > maxY {
			maxY = y1
		}
	}

	palette := color.Palette{
		color.NRGBA{0xfa, 0xfa, 0xfa, 0xff},
		color.NRGBA{0xf2, 0xf2, 0xf2, 0xff},
		color.NRGBA{0x10, 0x10, 0x10, 0xff},
		color.NRGBA{0xe0, 0xe0, 0xe0, 0xff},
		color.NRGBA{0x55, 0x55, 0x55, 0xff},
		color.NRGBA{0x33, 0x33, 0x33, 0xff},
	}

	type sized struct{ w, h int }
	keyframes := []sized{
		{960, 480},
		{1080, 540},
		{1200, 600},
		{1320, 660},
		{1440, 720},
	}
	// Build a smooth sweep: forward then reverse so the GIF loops cleanly.
	const steps = 8
	var sequence []sized
	for i := 0; i < len(keyframes)-1; i++ {
		a, b := keyframes[i], keyframes[i+1]
		for s := 0; s < steps; s++ {
			t := float64(s) / float64(steps)
			sequence = append(sequence, sized{
				w: a.w + int(float64(b.w-a.w)*t),
				h: a.h + int(float64(b.h-a.h)*t),
			})
		}
	}
	for i := len(keyframes) - 1; i > 0; i-- {
		a, b := keyframes[i], keyframes[i-1]
		for s := 0; s < steps; s++ {
			t := float64(s) / float64(steps)
			sequence = append(sequence, sized{
				w: a.w + int(float64(b.w-a.w)*t),
				h: a.h + int(float64(b.h-a.h)*t),
			})
		}
	}

	canvasW, canvasH := 1480, 760
	g := &gif.GIF{LoopCount: 0}
	for _, s := range sequence {
		canvas := image.NewPaletted(image.Rect(0, 0, canvasW, canvasH), palette)
		draw.Draw(canvas, canvas.Bounds(), image.NewUniform(palette[3]), image.Point{}, draw.Src)
		win := renderWindow(def, snap, 0, aabbs, minX, minY, maxX, maxY, s.w, s.h, palette)
		offset := image.Point{X: (canvasW - s.w) / 2, Y: (canvasH - s.h) / 2}
		dst := image.Rect(offset.X, offset.Y, offset.X+s.w, offset.Y+s.h)
		draw.Draw(canvas, dst, win, image.Point{}, draw.Src)
		drawRectBorder(canvas, dst.Inset(-1), palette[5])
		drawText(canvas, fmt.Sprintf("%dx%d", s.w, s.h), 12, 22, palette[2])

		g.Image = append(g.Image, canvas)
		g.Delay = append(g.Delay, 6) // 60 ms per frame
		g.Disposal = append(g.Disposal, gif.DisposalBackground)
	}

	f, err := os.Create(*out)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	if err := gif.EncodeAll(f, g); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("wrote %s (%d frames)\n", *out, len(g.Image))
}
