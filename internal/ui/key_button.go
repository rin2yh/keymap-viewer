// SPDX-License-Identifier: Apache-2.0

package ui

import (
	"image/color"
	"strings"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget"
)

// KeyButton renders a single physical key with its current label.
// It is purely presentational — there is no click handler. The whole
// viewer is read-only, so a key tap intentionally does nothing.
type KeyButton struct {
	guigui.DefaultWidget

	label string
}

// SetLabel updates the label drawn on this key.
func (k *KeyButton) SetLabel(label string) {
	if k.label != label {
		k.label = label
		guigui.RequestRedraw(k)
	}
}

// Label returns the current label.
func (k *KeyButton) Label() string {
	return k.label
}

func (k *KeyButton) WriteStateKey(w *guigui.StateKeyWriter) {
	w.WriteString(k.label)
}

func (k *KeyButton) Draw(context *guigui.Context, widgetBounds *guigui.WidgetBounds, dst *ebiten.Image) {
	bounds := widgetBounds.Bounds()
	if bounds.Empty() {
		return
	}

	bgFill := color.NRGBA{R: 0xf2, G: 0xf2, B: 0xf2, A: 0xff}
	border := color.NRGBA{R: 0x77, G: 0x77, B: 0x77, A: 0xff}
	textClr := color.NRGBA{R: 0x10, G: 0x10, B: 0x10, A: 0xff}
	if context.ColorMode() == ebiten.ColorModeDark {
		bgFill = color.NRGBA{R: 0x33, G: 0x33, B: 0x33, A: 0xff}
		border = color.NRGBA{R: 0x99, G: 0x99, B: 0x99, A: 0xff}
		textClr = color.NRGBA{R: 0xee, G: 0xee, B: 0xee, A: 0xff}
	}

	scale := float32(context.Scale())
	x := float32(bounds.Min.X)
	y := float32(bounds.Min.Y)
	w := float32(bounds.Dx())
	h := float32(bounds.Dy())

	vector.FillRect(dst, x, y, w, h, bgFill, false)
	vector.StrokeRect(dst, x, y, w, h, scale, border, false)

	if k.label == "" {
		return
	}
	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(bounds.Min.X+bounds.Dx()/2), float64(bounds.Min.Y+bounds.Dy()/2))
	op.ColorScale.ScaleWithColor(textClr)
	op.PrimaryAlign = text.AlignCenter
	op.SecondaryAlign = text.AlignCenter

	face := pickFontFace(context, bounds.Dx(), bounds.Dy(), k.label)
	if gtf, ok := face.(*text.GoTextFace); ok {
		op.LineSpacing = gtf.Size
	}
	text.Draw(dst, k.label, face, op)
}

// pickFontFace chooses a font size that scales with the cell while staying
// small enough to fit longer composite labels (e.g. "LCtl+Sft+/") and
// multi-line labels (Remap-style shifted-glyph stack like "?\n/").
//
// Width budget assumes the default font's average glyph advance is roughly
// 0.55 × font size. The longest-line advance must fit in 90% of the cell
// width, so size ≤ w × 0.9 ÷ (longest × 0.55). This lets short labels like
// "MO(0)" or "TO(0)" stay at the default size on a 1u cell instead of being
// uniformly shrunk by a fixed width factor.
func pickFontFace(context *guigui.Context, w, h int, label string) text.Face {
	base := basicwidget.FontSize(context)
	parts := strings.Split(label, "\n")
	lines := len(parts)
	longest := 0
	for _, line := range parts {
		if n := len(line); n > longest {
			longest = n
		}
	}
	if longest < 1 {
		longest = 1
	}

	const glyphAdvance = 0.55
	limitH := float64(h) / float64(lines) * 0.85
	limitW := float64(w) * 0.9 / (float64(longest) * glyphAdvance)

	size := base
	if limitH < size {
		size = limitH
	}
	if limitW < size {
		size = limitW
	}
	if size < 6 {
		size = 6
	}
	return faceForSize(size)
}

// faceCache memoises GoTextFace by size. Drawn keys re-enter pickFontFace
// every frame; without this, a fresh face is allocated per key per frame.
var (
	faceCacheMu sync.Mutex
	faceCache   = map[float64]*text.GoTextFace{}
)

func faceForSize(size float64) *text.GoTextFace {
	faceCacheMu.Lock()
	defer faceCacheMu.Unlock()
	if f, ok := faceCache[size]; ok {
		return f
	}
	f := &text.GoTextFace{
		Source: basicwidget.DefaultFaceSourceEntry().FaceSource,
		Size:   size,
	}
	faceCache[size] = f
	return f
}
