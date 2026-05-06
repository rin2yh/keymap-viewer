package keymap_test

import (
	"testing"

	"github.com/rin2yh/keymap-viewer/internal/keymap"
)

func TestLabel(t *testing.T) {
	tests := []struct {
		name string
		kc   uint16
		want string
	}{
		{"no-op", 0x0000, "✗"},
		{"transparent", 0x0001, "▽"},

		{"A", 0x0004, "A"},
		{"Z", 0x001D, "Z"},
		{"1", 0x001E, "1"},
		{"0", 0x0027, "0"},
		{"minus", 0x002D, "-"},

		{"BS", 0x002A, "BS"},
		{"Space", 0x002C, "Space"},
		{"Caps Lock", 0x0039, "Caps\nLock"},
		{"slash", 0x0038, "?\n/"},
		{"semicolon", 0x0033, ":\n;"},
		{"comma", 0x0036, "<\n,"},
		{"period", 0x0037, ">\n."},
		{"quote", 0x0034, "\"\n'"},

		{"Right", 0x004F, "→"},
		{"Left", 0x0050, "←"},
		{"Down", 0x0051, "↓"},
		{"Up", 0x0052, "↑"},

		{"LCtl", 0x00E0, "LCtl"},
		{"LSft", 0x00E1, "LSft"},
		{"RCtl", 0x00E4, "RCtl"},

		// Modern QMK keycode block: TO=0x5200, MO=0x5220, DF=0x5240,
		// TG=0x5260, OSL=0x5280, OSM=0x52A0, TT=0x52C0.
		{"TO(0)", 0x5200, "TO(0)"},
		{"TO(1)", 0x5201, "TO(1)"},
		{"MO(0)", 0x5220, "MO(0)"},
		{"MO(3)", 0x5223, "MO(3)"},
		{"TG(0)", 0x5260, "TG(0)"},
		{"OSM(LCtl)", 0x52A1, "OSM(LCtl)"},
		{"OSM(LCtl+LSft)", 0x52A3, "OSM(LCtl+LSft)"},
		{"TT(2)", 0x52C2, "TT(2)"},

		// LT(layer, kc): bits[11:8] = layer, bits[7:0] = base keycode.
		// 0x4104 = layer 1 with KC_A (0x04).
		{"LT1(A)", 0x4104, "LT1(A)"},

		{"M0", 0x7700, "M0"},
		{"M5", 0x7705, "M5"},

		// Audio / media / system / mouse keys; before this fix these fell
		// through to the hex fallback. Labels mirror Remap's keycode picker
		// strings with spaces converted to newlines for cap rendering.
		{"Mute", 0x00A8, "Audio\nMute"},
		{"VolUp", 0x00A9, "Audio\nVol +"},
		{"VolDown", 0x00AA, "Audio\nVol -"},
		{"Next", 0x00AB, "Next"},
		{"Play", 0x00AE, "Play"},
		{"Sleep", 0x00A6, "Sleep"},
		{"Mail", 0x00B1, "Mail"},
		{"BrightUp", 0x00BD, "Screen +"},
		{"MissionControl", 0x00C1, "Mission\nControl"},
		{"MouseUp", 0x00CD, "Mouse\n↑"},
		{"MouseBtn1", 0x00D1, "Mouse\nBtn1"},
		{"WheelUp", 0x00D9, "Mouse\nWh ↑"},
		{"MouseAcc0", 0x00DD, "Mouse\nAcc0"},

		// Mouse-key keycode wrapped in a modifier mask: the lookup must
		// recurse through the basic-table fix so the cap shows the named
		// mouse action instead of `…+0xCD`.
		{"LSft+MouseUp", 0x02CD, "LSft+Mouse\n↑"},

		{"unknown", 0xFFFF, "0xFFFF"},

		{"LSft+A", 0x0204, "LSft+A"},

		// MT(mod, kc) — Remap-style hold-tap rendering.
		// MT(MOD_LSFT, KC_A) = 0x2000 | (0x02 << 8) | 0x04 = 0x2204.
		{"*Shift", 0x2204, "*Shift"},
		// MT(MOD_LCTL, KC_Z) = 0x2000 | (0x01 << 8) | 0x1D = 0x211D.
		{"*Ctrl", 0x211D, "*Ctrl"},
		// MT(MOD_LGUI, KC_SPC) = 0x2000 | (0x08 << 8) | 0x2C = 0x282C.
		{"*Win", 0x282C, "*Win"},
		// MT(MOD_RALT, KC_SLASH) = 0x2000 | (0x14 << 8) | 0x38 = 0x3438.
		{"Alt*", 0x3438, "Alt*"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := keymap.Label(tc.kc)
			if got != tc.want {
				t.Errorf("Label(0x%04X) = %q, want %q", tc.kc, got, tc.want)
			}
		})
	}
}
