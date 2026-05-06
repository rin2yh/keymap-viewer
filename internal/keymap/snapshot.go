// SPDX-License-Identifier: Apache-2.0

package keymap

// Snapshot is a fully-materialised keymap read from a VIA device. Data is
// indexed as Data[layer][row][col] and is treated as immutable by the UI.
type Snapshot struct {
	Layers int
	Rows   int
	Cols   int
	Data   [][][]uint16
}

// NewSnapshot allocates the backing storage for the given dimensions.
func NewSnapshot(layers, rows, cols int) *Snapshot {
	d := make([][][]uint16, layers)
	for l := range d {
		layer := make([][]uint16, rows)
		for r := range layer {
			layer[r] = make([]uint16, cols)
		}
		d[l] = layer
	}
	return &Snapshot{Layers: layers, Rows: rows, Cols: cols, Data: d}
}

// Keycode returns the raw QMK keycode for the given (layer, row, col),
// or 0 if any index is out of range.
func (s *Snapshot) Keycode(layer, row, col int) uint16 {
	if s == nil {
		return 0
	}
	if layer < 0 || layer >= s.Layers {
		return 0
	}
	if row < 0 || row >= s.Rows {
		return 0
	}
	if col < 0 || col >= s.Cols {
		return 0
	}
	return s.Data[layer][row][col]
}
