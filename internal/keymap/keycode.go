package keymap

import (
	"fmt"
	"strings"
)

// Label converts a 16-bit QMK keycode to a short, human-readable token.
// Composite keycodes (LT, MO, TG, modifier-tap, mod-mask) are rendered
// recursively. Unknown keycodes fall back to the literal hex form.
func Label(kc uint16) string {
	switch {
	case kc == 0x0000:
		return "✗"
	case kc == 0x0001:
		return "▽"
	case kc < 0x00FF:
		if s, ok := basicKeycodes[kc]; ok {
			return s
		}
	}

	switch {
	case kc >= 0x00E0 && kc <= 0x00E7:
		return modifierKeycodes[kc-0x00E0]

	// Modifier-masked keycodes (mod + base, e.g. LSFT(KC_A)): 0x0100..0x1FFF.
	// Bits 0x1F00 hold the modifier mask; lower byte is the keycode.
	case kc >= 0x0100 && kc <= 0x1FFF:
		mods := uint8((kc >> 8) & 0x1F)
		base := kc & 0xFF
		return formatModMask(mods) + Label(base)

	// MT(mod, kc) — modifier-tap (hold = mod, tap = kc): 0x2000..0x3FFF.
	// Remap renders these as the modifier name with an asterisk indicating
	// the hold-tap nature: `*Shift` for left-side mods, `Alt*` for right-side.
	// The tap keycode is intentionally omitted to match Remap's display.
	case kc >= 0x2000 && kc <= 0x3FFF:
		mods := uint8((kc >> 8) & 0x1F)
		return formatMTMod(mods)

	// LT(layer, kc): 0x4000..0x4FFF. layer = (kc >> 8) & 0x0F
	case kc >= 0x4000 && kc <= 0x4FFF:
		layer := (kc >> 8) & 0x0F
		base := kc & 0xFF
		return fmt.Sprintf("LT%d(%s)", layer, Label(base))

	// Modern QMK / Remap layer & one-shot keycode ranges (32 layers, 5-bit
	// indices). These supersede the legacy 0x5100-block layout used by older
	// QMK builds. Order: TO → MO → DF → TG → OSL → OSM → TT.
	case kc >= 0x5200 && kc <= 0x521F:
		return fmt.Sprintf("TO(%d)", kc-0x5200)
	case kc >= 0x5220 && kc <= 0x523F:
		return fmt.Sprintf("MO(%d)", kc-0x5220)
	case kc >= 0x5240 && kc <= 0x525F:
		return fmt.Sprintf("DF(%d)", kc-0x5240)
	case kc >= 0x5260 && kc <= 0x527F:
		return fmt.Sprintf("TG(%d)", kc-0x5260)
	case kc >= 0x5280 && kc <= 0x529F:
		return fmt.Sprintf("OSL(%d)", kc-0x5280)
	case kc >= 0x52A0 && kc <= 0x52BF:
		return "OSM(" + strings.Join(modList(uint8(kc&0x1F)), "+") + ")"
	case kc >= 0x52C0 && kc <= 0x52DF:
		return fmt.Sprintf("TT(%d)", kc-0x52C0)

	// Tap-Dance: 0x5700..0x57FF
	case kc >= 0x5700 && kc <= 0x57FF:
		return fmt.Sprintf("TD(%d)", kc-0x5700)

	// Macros: VIA exposes M0..M15 at 0x7700..0x770F by convention.
	case kc >= 0x7700 && kc <= 0x77FF:
		return fmt.Sprintf("M%d", kc-0x7700)
	}

	return fmt.Sprintf("0x%04X", kc)
}

// formatMTMod renders a single MT(mod, kc) hold-side modifier the way Remap
// does: pure modifier name with an asterisk indicating tap-hold. Asterisk goes
// before for left-side mods (`*Shift`) and after for right-side (`Alt*`).
// Combinations of mods fall back to the generic mod-mask formatter.
func formatMTMod(mods uint8) string {
	right := mods&0x10 != 0
	var name string
	switch mods & 0x0F {
	case 0x01:
		name = "Ctrl"
	case 0x02:
		name = "Shift"
	case 0x04:
		name = "Alt"
	case 0x08:
		name = "Win"
	default:
		return formatModMask(mods) + "*"
	}
	if right {
		return name + "*"
	}
	return "*" + name
}

func formatModMask(mods uint8) string {
	list := modList(mods)
	if len(list) == 0 {
		return ""
	}
	// Trailing "+" so the mask can be prefixed onto a base label like "LSft+A".
	return strings.Join(list, "+") + "+"
}

// modList expands a 5-bit modifier mask into per-mod tokens (e.g. "LCtl",
// "RSft"). Bit 0x10 selects the right-hand side variants.
func modList(mods uint8) []string {
	if mods == 0 {
		return nil
	}
	prefix := "L"
	if mods&0x10 != 0 {
		prefix = "R"
	}
	var out []string
	if mods&0x01 != 0 {
		out = append(out, prefix+"Ctl")
	}
	if mods&0x02 != 0 {
		out = append(out, prefix+"Sft")
	}
	if mods&0x04 != 0 {
		out = append(out, prefix+"Alt")
	}
	if mods&0x08 != 0 {
		out = append(out, prefix+"Win")
	}
	return out
}

var modifierKeycodes = [...]string{
	"LCtl", "LSft", "LAlt", "LWin",
	"RCtl", "RSft", "RAlt", "RWin",
}

var basicKeycodes = map[uint16]string{
	0x0004: "A", 0x0005: "B", 0x0006: "C", 0x0007: "D",
	0x0008: "E", 0x0009: "F", 0x000A: "G", 0x000B: "H",
	0x000C: "I", 0x000D: "J", 0x000E: "K", 0x000F: "L",
	0x0010: "M", 0x0011: "N", 0x0012: "O", 0x0013: "P",
	0x0014: "Q", 0x0015: "R", 0x0016: "S", 0x0017: "T",
	0x0018: "U", 0x0019: "V", 0x001A: "W", 0x001B: "X",
	0x001C: "Y", 0x001D: "Z",

	0x001E: "1", 0x001F: "2", 0x0020: "3", 0x0021: "4",
	0x0022: "5", 0x0023: "6", 0x0024: "7", 0x0025: "8",
	0x0026: "9", 0x0027: "0",

	0x0028: "Enter", 0x0029: "Esc", 0x002A: "BS", 0x002B: "Tab",
	0x002C: "Space", 0x002D: "-", 0x002E: "=", 0x002F: "[",
	0x0030: "]", 0x0031: "\\", 0x0032: "#", 0x0033: ":\n;",
	0x0034: "\"\n'", 0x0035: "`", 0x0036: "<\n,", 0x0037: ">\n.",
	0x0038: "?\n/", 0x0039: "Caps\nLock",

	0x003A: "F1", 0x003B: "F2", 0x003C: "F3", 0x003D: "F4",
	0x003E: "F5", 0x003F: "F6", 0x0040: "F7", 0x0041: "F8",
	0x0042: "F9", 0x0043: "F10", 0x0044: "F11", 0x0045: "F12",

	0x0046: "PrtSc", 0x0047: "ScrLk", 0x0048: "Pause",
	0x0049: "Ins", 0x004A: "Home", 0x004B: "PgUp",
	0x004C: "Del", 0x004D: "End", 0x004E: "PgDn",
	0x004F: "→", 0x0050: "←", 0x0051: "↓", 0x0052: "↑",

	0x0053: "NumLk", 0x0054: "KP/", 0x0055: "KP*",
	0x0056: "KP-", 0x0057: "KP+", 0x0058: "KPEnt",
	0x0059: "KP1", 0x005A: "KP2", 0x005B: "KP3",
	0x005C: "KP4", 0x005D: "KP5", 0x005E: "KP6",
	0x005F: "KP7", 0x0060: "KP8", 0x0061: "KP9",
	0x0062: "KP0", 0x0063: "KP.",

	0x0064: "NUBS", 0x0065: "App", 0x0066: "Power", 0x0067: "KP=",

	0x0068: "F13", 0x0069: "F14", 0x006A: "F15", 0x006B: "F16",
	0x006C: "F17", 0x006D: "F18", 0x006E: "F19", 0x006F: "F20",
	0x0070: "F21", 0x0071: "F22", 0x0072: "F23", 0x0073: "F24",

	// Rare locking / alt-function keys; Remap labels are kept verbatim with
	// spaces converted to newlines so the auto-shrinking cap renderer can
	// stack tokens vertically — matches the existing "Caps\nLock" convention.
	0x0082: "Locking\nCaps\nLock",
	0x0083: "Locking\nNum\nLock",
	0x0084: "Locking\nScroll\nLock",
	0x0085: "KP,",
	0x0086: "Num\n=\nAS400",

	0x0087: "INT1", 0x0088: "INT2", 0x0089: "INT3",
	0x008A: "INT4", 0x008B: "INT5", 0x008C: "INT6",
	0x008D: "INT7", 0x008E: "INT8", 0x008F: "INT9",
	0x0090: "LANG1", 0x0091: "LANG2", 0x0092: "LANG3",
	0x0093: "LANG4", 0x0094: "LANG5", 0x0095: "LANG6",
	0x0096: "LANG7", 0x0097: "LANG8", 0x0098: "LANG9",

	0x0099: "Alt\nErase", 0x009A: "SysReq", 0x009B: "Cancel",
	0x009C: "Clear", 0x009D: "Prior", 0x009E: "Return",
	0x009F: "Separator",
	0x00A0: "Out", 0x00A1: "Oper",
	0x00A2: "Clear/\nAgain", 0x00A3: "CrSel/\nProps", 0x00A4: "ExSel",

	// System power keys.
	0x00A5: "System\nPower\nDown", 0x00A6: "Sleep", 0x00A7: "Wake",

	// Audio / media keys.
	0x00A8: "Audio\nMute",
	0x00A9: "Audio\nVol +", 0x00AA: "Audio\nVol -",
	0x00AB: "Next", 0x00AC: "Previous",
	0x00AD: "Media\nStop", 0x00AE: "Play", 0x00AF: "Select",
	0x00B0: "Eject",

	// Web / launcher / system shortcut keys.
	0x00B1: "Mail", 0x00B2: "Calculator", 0x00B3: "My\nComputer",
	0x00B4: "WWW\nSearch", 0x00B5: "WWW\nHome",
	0x00B6: "WWW\nBack", 0x00B7: "WWW\nForward",
	0x00B8: "WWW\nStop", 0x00B9: "WWW\nRefresh",
	0x00BA: "WWW\nFavorite",
	0x00BB: "Fast\nForward", 0x00BC: "Rewind",
	0x00BD: "Screen +", 0x00BE: "Screen -",
	0x00BF: "Open\nControl\nPanel",
	0x00C0: "Assistant", 0x00C1: "Mission\nControl", 0x00C2: "Launchpad",

	// Mouse keys.
	0x00CD: "Mouse\n↑", 0x00CE: "Mouse\n↓",
	0x00CF: "Mouse\n←", 0x00D0: "Mouse\n→",
	0x00D1: "Mouse\nBtn1", 0x00D2: "Mouse\nBtn2",
	0x00D3: "Mouse\nBtn3", 0x00D4: "Mouse\nBtn4",
	0x00D5: "Mouse\nBtn5", 0x00D6: "Mouse\nBtn6",
	0x00D7: "Mouse\nBtn7", 0x00D8: "Mouse\nBtn8",
	0x00D9: "Mouse\nWh ↑", 0x00DA: "Mouse\nWh ↓",
	0x00DB: "Mouse\nWh ←", 0x00DC: "Mouse\nWh →",
	0x00DD: "Mouse\nAcc0", 0x00DE: "Mouse\nAcc1", 0x00DF: "Mouse\nAcc2",
}
