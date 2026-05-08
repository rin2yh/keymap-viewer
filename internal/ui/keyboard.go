package ui

import (
	"image"
	"math"

	"github.com/guigui-gui/guigui"

	"github.com/rin2yh/keymap-viewer/internal/keymap"
)

// keyUnitPx is the on-screen pixel size of a 1u key cell BEFORE the scale
// factor — the actual unit is unitPx * context.Scale(). It is also the
// natural / minimum size used when the available bounds are larger than the
// keymap or when fitting cannot be computed.
const keyUnitPx = 56

// keyGapRatio is the inset around each key as a fraction of the unit size,
// so the gap shrinks/grows together with the keys when the window resizes.
// 4/56 matches the previous fixed 4px gap at the default 56px unit.
const keyGapRatio = 4.0 / 56.0

// keyMinUnitPx caps how small a 1u cell may shrink before key labels stop
// being legible.
const keyMinUnitPx = 18

// Keyboard is the custom widget that lays out the Corne v4 Chocolate keys at
// VIA-defined absolute positions. guigui itself does not yet support absolute
// (overlapping/rotated) positioning of children, so all positioning is done
// in this widget's Layout method via direct ChildLayouter.LayoutWidget calls.
type Keyboard struct {
	guigui.DefaultWidget

	def      *keymap.Definition
	snapshot *keymap.Snapshot
	layer    int

	buttons []KeyButton

	// keyAABBs caches per-key axis-aligned bounding boxes in unit space
	// (1.0 = one keycap). Recomputed on SetDefinition; reused on every Layout.
	keyAABBs []aabb
	// minX, minY, maxX, maxY is the layout's overall AABB in unit space,
	// cached so Layout/Measure don't refold over every key per frame.
	minX, minY, maxX, maxY float64
}

type aabb struct {
	x0, y0, x1, y1 float64
}

// SetDefinition wires the static physical layout. Must be called before the
// first Build pass.
func (k *Keyboard) SetDefinition(def *keymap.Definition) {
	k.def = def
	k.buttons = make([]KeyButton, len(def.Keys))
	k.keyAABBs = make([]aabb, len(def.Keys))
	if len(def.Keys) == 0 {
		k.minX, k.minY, k.maxX, k.maxY = 0, 0, 0, 0
		return
	}
	for i, key := range def.Keys {
		x0, y0, x1, y1 := keyAABBUnits(key)
		k.keyAABBs[i] = aabb{x0, y0, x1, y1}
		if i == 0 {
			k.minX, k.minY, k.maxX, k.maxY = x0, y0, x1, y1
			continue
		}
		k.minX = min(k.minX, x0)
		k.minY = min(k.minY, y0)
		k.maxX = max(k.maxX, x1)
		k.maxY = max(k.maxY, y1)
	}
}

// SetSnapshot replaces the data displayed on the keys.
func (k *Keyboard) SetSnapshot(snap *keymap.Snapshot) {
	if k.snapshot == snap {
		return
	}
	k.snapshot = snap
	guigui.RequestRebuild(k)
}

// SetLayer changes the active layer index.
func (k *Keyboard) SetLayer(layer int) {
	if layer < 0 {
		layer = 0
	}
	if k.layer != layer {
		k.layer = layer
		guigui.RequestRebuild(k)
	}
}

// Layer returns the currently displayed layer.
func (k *Keyboard) Layer() int {
	return k.layer
}

func (k *Keyboard) WriteStateKey(w *guigui.StateKeyWriter) {
	w.WriteUint64(uint64(k.layer))
	if k.snapshot != nil {
		w.WriteUint64(uint64(k.snapshot.Layers))
		w.WriteUint64(uint64(k.snapshot.Rows))
		w.WriteUint64(uint64(k.snapshot.Cols))
	}
}

func (k *Keyboard) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	if k.def == nil {
		return nil
	}
	for i, key := range k.def.Keys {
		btn := &k.buttons[i]
		var label string
		if k.snapshot != nil {
			label = keymap.Label(k.snapshot.Keycode(k.layer, key.Row, key.Col))
		}
		btn.SetLabel(label)
		adder.AddWidget(btn)
	}
	return nil
}

func (k *Keyboard) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	if k.def == nil {
		return
	}
	bounds := widgetBounds.Bounds()
	unit := k.fitUnit(context, bounds)
	gap := int(math.Round(float64(unit) * keyGapRatio))
	totalW := (k.maxX - k.minX) * float64(unit)
	totalH := (k.maxY - k.minY) * float64(unit)
	offX := bounds.Min.X + (bounds.Dx()-int(totalW))/2 - int(k.minX*float64(unit))
	offY := bounds.Min.Y + (bounds.Dy()-int(totalH))/2 - int(k.minY*float64(unit))

	for i, a := range k.keyAABBs {
		rect := image.Rect(
			offX+int(a.x0*float64(unit))+gap,
			offY+int(a.y0*float64(unit))+gap,
			offX+int(a.x1*float64(unit))-gap,
			offY+int(a.y1*float64(unit))-gap,
		)
		layouter.LayoutWidget(&k.buttons[i], rect)
	}
}

// fitUnit returns the per-unit pixel size that fits the keymap into bounds.
// If bounds are larger than the natural size, the unit grows so the keymap
// fills the available space; if smaller, it shrinks down to keyMinUnitPx so
// labels remain legible.
func (k *Keyboard) fitUnit(context *guigui.Context, bounds image.Rectangle) int {
	widthU := k.maxX - k.minX
	heightU := k.maxY - k.minY
	if widthU <= 0 || heightU <= 0 || bounds.Dx() <= 0 || bounds.Dy() <= 0 {
		return unitPixels(context)
	}
	fit := math.Min(float64(bounds.Dx())/widthU, float64(bounds.Dy())/heightU)
	minPx := math.Round(keyMinUnitPx * context.Scale())
	if fit < minPx {
		fit = minPx
	}
	return int(math.Floor(fit))
}

func (k *Keyboard) Measure(context *guigui.Context, _ guigui.Constraints) image.Point {
	if k.def == nil {
		return image.Point{}
	}
	unit := unitPixels(context)
	w := int((k.maxX - k.minX) * float64(unit))
	h := int((k.maxY - k.minY) * float64(unit))
	return image.Pt(w, h)
}

// keyAABBUnits returns the (axis-aligned) bounding box of the key in keymap
// "units" — 1u = one keycap. Rotated keys (e.g. Corne thumb cluster) have
// their CENTER point rotated around (rx, ry); the resulting cell is the
// same w × h sized rectangle drawn axis-aligned at that rotated centre.
//
// Drawing the cell axis-aligned (rather than tracking the actual rotated
// quad) costs us a small visual fidelity hit on the thumb keys but keeps
// neighbouring cells from overlapping their AABBs — the literal rotated
// AABB is up to ~40% larger than the keycap and would crash into adjacent
// keys.
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

	// Rotate the centre of the key around (rx, ry).
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

func unitPixels(context *guigui.Context) int {
	return int(math.Round(float64(keyUnitPx) * context.Scale()))
}
