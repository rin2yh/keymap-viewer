package keymap

import (
	"testing"
)

func TestLabel(t *testing.T) {
	// modMask has no prefix bit — the range is identified by mods != 0,
	// unlike modTap/layerTap whose range minima ARE prefix bits.
	modMask := func(mods uint8, base uint16) uint16 {
		return uint16(mods)<<modBitsShift | (base & baseKCMask)
	}
	modTap := func(mods uint8, base uint16) uint16 {
		return modTapMin | uint16(mods)<<modBitsShift | (base & baseKCMask)
	}
	layerTap := func(layer uint16, base uint16) uint16 {
		return layerTapMin | (layer&layerFieldMask)<<modBitsShift | (base & baseKCMask)
	}

	tests := []struct {
		name string
		kc   uint16
		want string
	}{
		{"no-op", kcNoOp, labelNoOp},
		{"transparent", kcTransparent, labelTransparent},

		{"A", kcA, "A"},
		{"Z", kcZ, "Z"},
		{"1", kc1, "1"},
		{"0", kc0, "0"},
		{"minus", kcMinus, "-"},

		{"BS", kcBackspace, "BS"},
		{"Space", kcSpace, "Space"},
		{"Caps Lock", kcCapsLock, "Caps\nLock"},
		{"slash", kcSlash, "?\n/"},
		{"semicolon", kcSemicolon, ":\n;"},
		{"comma", kcComma, "<\n,"},
		{"period", kcDot, ">\n."},
		{"quote", kcQuote, "\"\n'"},

		{"Right", kcRight, "→"},
		{"Left", kcLeft, "←"},
		{"Down", kcDown, "↓"},
		{"Up", kcUp, "↑"},

		{"LCtl", modifierMin, "LCtl"},
		{"LSft", modifierMin + 1, "LSft"},
		{"RCtl", modifierMin + 4, "RCtl"},

		{"TO(0)", layerToBase, "TO(0)"},
		{"TO(1)", layerToBase + 1, "TO(1)"},
		{"MO(0)", layerMoBase, "MO(0)"},
		{"MO(3)", layerMoBase + 3, "MO(3)"},
		{"TG(0)", layerTgBase, "TG(0)"},
		{"OSM(LCtl)", layerOsmBase | uint16(modBitCtrl), "OSM(LCtl)"},
		{"OSM(LCtl+LSft)", layerOsmBase | uint16(modBitCtrl|modBitShift), "OSM(LCtl+LSft)"},
		{"TT(2)", layerTtBase + 2, "TT(2)"},

		{"LT1(A)", layerTap(1, kcA), "LT1(A)"},

		{"M0", macroMin, "M0"},
		{"M5", macroMin + 5, "M5"},

		{"Mute", kcAudioMute, "Audio\nMute"},
		{"VolUp", kcAudioVolUp, "Audio\nVol +"},
		{"VolDown", kcAudioVolDown, "Audio\nVol -"},
		{"Next", kcMediaNext, "Next"},
		{"Play", kcMediaPlay, "Play"},
		{"Sleep", kcSystemSleep, "Sleep"},
		{"Mail", kcMail, "Mail"},
		{"BrightUp", kcBrightnessUp, "Screen +"},
		{"MissionControl", kcMissionControl, "Mission\nControl"},
		{"MouseUp", kcMouseUp, "Mouse\n↑"},
		{"MouseBtn1", kcMouseBtn1, "Mouse\nBtn1"},
		{"WheelUp", kcMouseWheelUp, "Mouse\nWh ↑"},
		{"MouseAcc0", kcMouseAccel0, "Mouse\nAcc0"},

		// Mod-masked composite must recurse through basicKeycodes so the
		// rendered label still names the mouse action.
		{"LSft+MouseUp", modMask(modBitShift, kcMouseUp), "LSft+Mouse\n↑"},

		{"unknown", 0xFFFF, "0xFFFF"},

		{"LSft+A", modMask(modBitShift, kcA), "LSft+A"},

		// MT asterisk position differs by side: prefix for left, suffix for right.
		{"*Shift", modTap(modBitShift, kcA), "*Shift"},
		{"*Ctrl", modTap(modBitCtrl, kcZ), "*Ctrl"},
		{"*Win", modTap(modBitWin, kcSpace), "*Win"},
		{"Alt*", modTap(modBitRight|modBitAlt, kcSlash), "Alt*"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := Label(tc.kc)
			if got != tc.want {
				t.Errorf("Label(0x%04X) = %q, want %q", tc.kc, got, tc.want)
			}
		})
	}
}
