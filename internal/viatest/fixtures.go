package viatest

import "github.com/rin2yh/keymap-viewer/internal/keymap"

// SampleSnapshot returns a deterministic 4-layer snapshot sized to match
// the embedded Corne v4 definition (8 rows × 7 cols). Each layer uses a
// distinct keycode family so that label rendering exercises a different
// branch of keymap.Label per layer:
//
//   - layer 0: basic keycodes (A..Z, digits, symbols)
//   - layer 1: shift-modified keycodes (mod-mask range)
//   - layer 2: LT1(base) layer-tap keycodes
//   - layer 3: transparent (▽) everywhere
//
// The pattern is used by E2E goldens; changing it requires regenerating
// e2e/testdata/golden/.
func SampleSnapshot() *keymap.Snapshot {
	const layers, rows, cols = 4, 8, 7
	snap := keymap.NewSnapshot(layers, rows, cols)
	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			// Cycle through 0x0004..0x0063 so labels include letters,
			// digits, navigation, and keypad keys.
			base := uint16(0x0004 + uint16(r*cols+c)%0x60)
			snap.Data[0][r][c] = base
			snap.Data[1][r][c] = 0x0200 | base // LSFT(base)
			snap.Data[2][r][c] = 0x4100 | base // LT1(base)
			snap.Data[3][r][c] = 0x0001        // transparent
		}
	}
	return snap
}
