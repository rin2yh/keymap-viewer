// SPDX-License-Identifier: Apache-2.0

package ui

import (
	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget"
)

// Header is the title strip across the top of the window. It shows the
// connection status (or any prevailing error) and a Reload button that
// re-fetches the keymap from the device.
type Header struct {
	guigui.DefaultWidget

	title  basicwidget.Text
	status basicwidget.Text
	reload basicwidget.Button

	statusValue string
	titleValue  string
	onReload    func(context *guigui.Context)

	items []guigui.LinearLayoutItem
}

// SetTitle updates the leading text (typically the keyboard name).
func (h *Header) SetTitle(s string) {
	if h.titleValue == s {
		return
	}
	h.titleValue = s
}

// SetStatus updates the trailing status string (typically connection state).
func (h *Header) SetStatus(s string) {
	if h.statusValue == s {
		return
	}
	h.statusValue = s
}

// OnReload registers a click handler for the Reload button.
func (h *Header) OnReload(f func(context *guigui.Context)) {
	h.onReload = f
}

func (h *Header) WriteStateKey(w *guigui.StateKeyWriter) {
	w.WriteString(h.titleValue)
	w.WriteString(h.statusValue)
}

func (h *Header) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	h.title.SetValue(h.titleValue)
	h.title.SetBold(true)
	h.title.SetVerticalAlign(basicwidget.VerticalAlignMiddle)

	h.status.SetValue(h.statusValue)
	h.status.SetVerticalAlign(basicwidget.VerticalAlignMiddle)

	h.reload.SetText("Reload")
	if h.onReload != nil {
		cb := h.onReload
		h.reload.OnUp(cb)
	}

	adder.AddWidget(&h.title)
	adder.AddWidget(&h.status)
	adder.AddWidget(&h.reload)
	return nil
}

func (h *Header) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	u := basicwidget.UnitSize(context)
	h.items = h.items[:0]
	h.items = append(h.items,
		guigui.LinearLayoutItem{Widget: &h.title, Size: guigui.FixedSize(8 * u)},
		guigui.LinearLayoutItem{Widget: &h.status, Size: guigui.FlexibleSize(1)},
		guigui.LinearLayoutItem{Widget: &h.reload, Size: guigui.FixedSize(5 * u)},
	)
	(guigui.LinearLayout{
		Direction: guigui.LayoutDirectionHorizontal,
		Items:     h.items,
		Gap:       u / 2,
		Padding: guigui.Padding{
			Start: u / 2,
			End:   u / 2,
		},
	}).LayoutWidgets(context, widgetBounds.Bounds(), layouter)
}
