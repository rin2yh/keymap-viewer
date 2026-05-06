// SPDX-License-Identifier: Apache-2.0

package ui

import (
	"fmt"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget"
)

// LayerTabs is a vertical segmented control that selects the active
// keymap layer. The number of tabs is driven by the snapshot's layer count.
type LayerTabs struct {
	guigui.DefaultWidget

	control basicwidget.SegmentedControl[int]
	count   int
	current int

	onChange func(context *guigui.Context, layer int)
	items    []basicwidget.SegmentedControlItem[int]
}

// SetCount sets how many layer tabs to display.
func (t *LayerTabs) SetCount(n int) {
	if n < 0 {
		n = 0
	}
	if t.count == n {
		return
	}
	t.count = n
	if t.current >= n {
		t.current = 0
	}
}

// SetCurrent updates the selected layer.
func (t *LayerTabs) SetCurrent(layer int) {
	if layer < 0 || layer >= t.count {
		return
	}
	t.current = layer
}

// OnChange registers a layer-change callback.
func (t *LayerTabs) OnChange(f func(context *guigui.Context, layer int)) {
	t.onChange = f
}

func (t *LayerTabs) WriteStateKey(w *guigui.StateKeyWriter) {
	w.WriteUint64(uint64(t.count))
	w.WriteUint64(uint64(t.current))
}

func (t *LayerTabs) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	t.items = t.items[:0]
	for i := 0; i < t.count; i++ {
		t.items = append(t.items, basicwidget.SegmentedControlItem[int]{
			Text:  fmt.Sprintf("L%d", i),
			Value: i,
		})
	}
	t.control.SetDirection(basicwidget.SegmentedControlDirectionVertical)
	t.control.SetItems(t.items)
	if t.count > 0 {
		t.control.SelectItemByValue(t.current)
	}
	t.control.OnItemSelected(func(context *guigui.Context, index int) {
		if index < 0 || index >= t.count {
			return
		}
		if t.current == index {
			return
		}
		t.current = index
		if t.onChange != nil {
			t.onChange(context, index)
		}
	})
	adder.AddWidget(&t.control)
	return nil
}

func (t *LayerTabs) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	layouter.LayoutWidget(&t.control, widgetBounds.Bounds())
}
