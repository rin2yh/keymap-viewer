// SPDX-License-Identifier: Apache-2.0

package ui

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget"

	"github.com/yuuki/keymap-viewer/internal/keymap"
	"github.com/yuuki/keymap-viewer/internal/via"
)

// ClientOpener returns an opened VIA client. ui.Root calls it on every
// snapshot fetch.
type ClientOpener func() (*via.ReadOnlyClient, error)

// Root is the top-level widget for the read-only Remap viewer. It places the
// Header at the top, the vertical LayerTabs along the left edge, and the
// Keyboard filling the rest. It fetches the initial snapshot from the device
// on first build, and reloads on demand.
type Root struct {
	guigui.DefaultWidget

	background basicwidget.Background
	header     Header
	tabs       LayerTabs
	keyboard   Keyboard

	def    *keymap.Definition
	opener ClientOpener

	mu       sync.Mutex
	snapshot *keymap.Snapshot
	status   string

	loading       atomic.Bool
	initialFetch  atomic.Bool
	pendingResult atomic.Pointer[fetchResult]
}

type fetchResult struct {
	snap *keymap.Snapshot
	err  error
}

// NewRoot constructs a Root for the given keyboard definition. The opener
// is invoked on every snapshot fetch; production callers pass via.Open.
func NewRoot(def *keymap.Definition, opener ClientOpener) *Root {
	r := &Root{def: def, opener: opener}
	r.keyboard.SetDefinition(def)
	r.header.SetTitle(def.Name)
	r.header.SetStatus("Connecting…")
	r.header.OnReload(func(context *guigui.Context) {
		r.startFetch()
	})
	r.tabs.OnChange(func(context *guigui.Context, layer int) {
		r.keyboard.SetLayer(layer)
	})
	return r
}

// startFetch kicks off a background snapshot fetch. Returns immediately; the
// result is consumed by the next Tick.
func (r *Root) startFetch() {
	if !r.loading.CompareAndSwap(false, true) {
		return
	}
	r.mu.Lock()
	r.status = "Reading keymap…"
	r.mu.Unlock()
	go func() {
		snap, err := readKeymap(r.opener, r.def)
		r.pendingResult.Store(&fetchResult{snap: snap, err: err})
	}()
}

func readKeymap(open ClientOpener, def *keymap.Definition) (*keymap.Snapshot, error) {
	client, err := open()
	if err != nil {
		return nil, err
	}
	defer client.Close()
	return via.FetchSnapshot(client, def.Matrix.Rows, def.Matrix.Cols)
}

func (r *Root) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	if !r.initialFetch.Swap(true) {
		r.startFetch()
	}

	r.mu.Lock()
	snap := r.snapshot
	status := r.status
	r.mu.Unlock()

	r.header.SetStatus(status)

	if snap != nil {
		r.tabs.SetCount(snap.Layers)
		r.keyboard.SetSnapshot(snap)
	} else {
		r.tabs.SetCount(0)
	}

	adder.AddWidget(&r.background)
	adder.AddWidget(&r.header)
	adder.AddWidget(&r.tabs)
	adder.AddWidget(&r.keyboard)
	return nil
}

func (r *Root) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	bounds := widgetBounds.Bounds()
	layouter.LayoutWidget(&r.background, bounds)

	u := basicwidget.UnitSize(context)
	pad := u / 2
	gap := u / 2
	tabsWidth := 3 * u

	inner := bounds
	inner.Min.X += pad
	inner.Min.Y += pad
	inner.Max.X -= pad
	inner.Max.Y -= pad

	headerBounds := inner
	headerBounds.Max.Y = inner.Min.Y + 2*u

	bodyBounds := inner
	bodyBounds.Min.Y = headerBounds.Max.Y + gap

	tabsBounds := bodyBounds
	tabsBounds.Max.X = bodyBounds.Min.X + tabsWidth

	keyboardBounds := bodyBounds
	keyboardBounds.Min.X = tabsBounds.Max.X + gap

	layouter.LayoutWidget(&r.header, headerBounds)
	layouter.LayoutWidget(&r.tabs, tabsBounds)
	layouter.LayoutWidget(&r.keyboard, keyboardBounds)
}

func (r *Root) Tick(context *guigui.Context, _ *guigui.WidgetBounds) error {
	if res := r.pendingResult.Swap(nil); res != nil {
		r.mu.Lock()
		if res.err != nil {
			r.status = fmt.Sprintf("Error: %v", res.err)
		} else {
			r.snapshot = res.snap
			r.status = fmt.Sprintf("Connected — %d layers × %d×%d", res.snap.Layers, res.snap.Rows, res.snap.Cols)
		}
		r.mu.Unlock()
		r.loading.Store(false)
		guigui.RequestRebuild(r)
	}
	return nil
}

func (r *Root) WriteStateKey(w *guigui.StateKeyWriter) {
	r.mu.Lock()
	w.WriteString(r.status)
	hasSnap := r.snapshot != nil
	r.mu.Unlock()
	w.WriteBool(hasSnap)
}
