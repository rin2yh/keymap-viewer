package keymap

import (
	"fmt"
	"strings"
)

// QMK keycode space layout. Each composite block is identified by an
// inclusive start/end pair so case bodies don't carry the magic numbers.
const (
	kcNoOp        uint16 = 0x0000 // KC_NO — unassigned
	kcTransparent uint16 = 0x0001 // KC_TRNS — fall through to lower layer

	// Modifier-only keycodes (LCtl..RWin), 8-wide.
	modifierMin uint16 = 0x00E0 // KC_LCTL
	modifierMax uint16 = 0x00E7 // KC_RGUI

	// Modifier-masked keycodes (mod + base, e.g. LSFT(KC_A)).
	// Bits 0x1F00 hold the modifier mask; lower byte is the base keycode.
	modMaskMin uint16 = 0x0100
	modMaskMax uint16 = 0x1FFF

	// MT(mod, kc) — modifier-tap. Same bit layout as modMask; the tap base
	// is intentionally dropped from the rendered label to match Remap.
	modTapMin uint16 = 0x2000
	modTapMax uint16 = 0x3FFF

	// LT(layer, kc) — layer-tap. Bits 0x0F00 are the 4-bit layer index;
	// lower byte is the base keycode.
	layerTapMin uint16 = 0x4000
	layerTapMax uint16 = 0x4FFF

	// Modern QMK layer-action block (32 layers per sub-range, 6 sub-ranges).
	// Order: TO → MO → DF → TG → OSL → OSM → TT.
	layerActionBase uint16 = 0x5200
	layerBlockSize  uint16 = 0x0020
	layerToBase     uint16 = layerActionBase + layerBlockSize*0
	layerMoBase     uint16 = layerActionBase + layerBlockSize*1
	layerDfBase     uint16 = layerActionBase + layerBlockSize*2
	layerTgBase     uint16 = layerActionBase + layerBlockSize*3
	layerOslBase    uint16 = layerActionBase + layerBlockSize*4
	layerOsmBase    uint16 = layerActionBase + layerBlockSize*5
	layerTtBase     uint16 = layerActionBase + layerBlockSize*6

	// Tap-Dance.
	tapDanceMin uint16 = 0x5700
	tapDanceMax uint16 = 0x57FF

	// VIA macro range — M0..M255 by convention.
	macroMin uint16 = 0x7700
	macroMax uint16 = 0x77FF

	// Bit-field layout shared by mod-mask, mod-tap, and layer-tap composites.
	modBitsShift   uint16 = 8
	baseKCMask     uint16 = 0x00FF // low byte holds the wrapped base keycode
	layerFieldMask uint16 = 0x000F // LT() layer index occupies 4 bits
)

// Modifier mask bit layout (5 bits = 4 mod bits + 1 right-side flag).
const (
	modBitCtrl  uint8 = 0x01
	modBitShift uint8 = 0x02
	modBitAlt   uint8 = 0x04
	modBitWin   uint8 = 0x08
	modBitRight uint8 = 0x10

	modCoreMask uint8 = modBitCtrl | modBitShift | modBitAlt | modBitWin
	modAllMask  uint8 = modCoreMask | modBitRight
)

// Display tokens for the special control keycodes and the trailing
// fallback. Centralised so tests can match against the same strings the
// renderer emits.
const (
	labelNoOp        = "✗"
	labelTransparent = "▽"
	hexLabelFormat   = "0x%04X"
)

// Label converts a 16-bit QMK keycode to a short, human-readable token.
// Composite keycodes (LT, MO, TG, modifier-tap, mod-mask) are rendered
// recursively. Unknown keycodes fall back to the literal hex form.
func Label(kc uint16) string {
	switch kc {
	case kcNoOp:
		return labelNoOp
	case kcTransparent:
		return labelTransparent
	}
	if s, ok := basicKeycodes[kc]; ok {
		return s
	}

	switch {
	case kc >= modifierMin && kc <= modifierMax:
		return modifierKeycodes[kc-modifierMin]

	case kc >= modMaskMin && kc <= modMaskMax:
		mods := uint8(kc>>modBitsShift) & modAllMask
		base := kc & baseKCMask
		return formatModMask(mods) + Label(base)

	case kc >= modTapMin && kc <= modTapMax:
		mods := uint8(kc>>modBitsShift) & modAllMask
		return formatMTMod(mods)

	case kc >= layerTapMin && kc <= layerTapMax:
		layer := (kc >> modBitsShift) & layerFieldMask
		base := kc & baseKCMask
		return fmt.Sprintf("LT%d(%s)", layer, Label(base))

	case inLayerBlock(kc, layerToBase):
		return fmt.Sprintf("TO(%d)", kc-layerToBase)
	case inLayerBlock(kc, layerMoBase):
		return fmt.Sprintf("MO(%d)", kc-layerMoBase)
	case inLayerBlock(kc, layerDfBase):
		return fmt.Sprintf("DF(%d)", kc-layerDfBase)
	case inLayerBlock(kc, layerTgBase):
		return fmt.Sprintf("TG(%d)", kc-layerTgBase)
	case inLayerBlock(kc, layerOslBase):
		return fmt.Sprintf("OSL(%d)", kc-layerOslBase)
	case inLayerBlock(kc, layerOsmBase):
		return "OSM(" + strings.Join(modList(uint8(kc)&modAllMask), "+") + ")"
	case inLayerBlock(kc, layerTtBase):
		return fmt.Sprintf("TT(%d)", kc-layerTtBase)

	case kc >= tapDanceMin && kc <= tapDanceMax:
		return fmt.Sprintf("TD(%d)", kc-tapDanceMin)

	case kc >= macroMin && kc <= macroMax:
		return fmt.Sprintf("M%d", kc-macroMin)
	}

	return fmt.Sprintf(hexLabelFormat, kc)
}

// inLayerBlock reports whether kc falls in the [base, base+layerBlockSize)
// sub-range of the modern layer-action block.
func inLayerBlock(kc, base uint16) bool {
	return kc >= base && kc < base+layerBlockSize
}

// formatMTMod renders a single MT(mod, kc) hold-side modifier the way Remap
// does: pure modifier name with an asterisk indicating tap-hold. Asterisk goes
// before for left-side mods (`*Shift`) and after for right-side (`Alt*`).
// Combinations of mods fall back to the generic mod-mask formatter.
func formatMTMod(mods uint8) string {
	right := mods&modBitRight != 0
	var name string
	switch mods & modCoreMask {
	case modBitCtrl:
		name = "Ctrl"
	case modBitShift:
		name = "Shift"
	case modBitAlt:
		name = "Alt"
	case modBitWin:
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
// "RSft"). The right-side flag selects the right-hand variants.
func modList(mods uint8) []string {
	if mods == 0 {
		return nil
	}
	prefix := "L"
	if mods&modBitRight != 0 {
		prefix = "R"
	}
	var out []string
	if mods&modBitCtrl != 0 {
		out = append(out, prefix+"Ctl")
	}
	if mods&modBitShift != 0 {
		out = append(out, prefix+"Sft")
	}
	if mods&modBitAlt != 0 {
		out = append(out, prefix+"Alt")
	}
	if mods&modBitWin != 0 {
		out = append(out, prefix+"Win")
	}
	return out
}

var modifierKeycodes = [...]string{
	"LCtl", "LSft", "LAlt", "LWin",
	"RCtl", "RSft", "RAlt", "RWin",
}

// Named constants for the basic HID-usage-page subset used in
// basicKeycodes. Defined here (not just inline as map keys) so white-box
// tests can drive Label() by name and code maintenance doesn't require
// reasoning about magic hex values.
const (
	kcA uint16 = 0x0004 + iota
	kcB
	kcC
	kcD
	kcE
	kcF
	kcG
	kcH
	kcI
	kcJ
	kcK
	kcL
	kcM
	kcN
	kcO
	kcP
	kcQ
	kcR
	kcS
	kcT
	kcU
	kcV
	kcW
	kcX
	kcY
	kcZ
)

const (
	kc1 uint16 = 0x001E + iota
	kc2
	kc3
	kc4
	kc5
	kc6
	kc7
	kc8
	kc9
	kc0
)

const (
	kcEnter        uint16 = 0x0028
	kcEscape       uint16 = 0x0029
	kcBackspace    uint16 = 0x002A
	kcTab          uint16 = 0x002B
	kcSpace        uint16 = 0x002C
	kcMinus        uint16 = 0x002D
	kcEqual        uint16 = 0x002E
	kcLeftBracket  uint16 = 0x002F
	kcRightBracket uint16 = 0x0030
	kcBackslash    uint16 = 0x0031
	kcNonUSHash    uint16 = 0x0032
	kcSemicolon    uint16 = 0x0033
	kcQuote        uint16 = 0x0034
	kcGrave        uint16 = 0x0035
	kcComma        uint16 = 0x0036
	kcDot          uint16 = 0x0037
	kcSlash        uint16 = 0x0038
	kcCapsLock     uint16 = 0x0039
)

const (
	kcF1 uint16 = 0x003A + iota
	kcF2
	kcF3
	kcF4
	kcF5
	kcF6
	kcF7
	kcF8
	kcF9
	kcF10
	kcF11
	kcF12
)

const (
	kcPrintScreen uint16 = 0x0046
	kcScrollLock  uint16 = 0x0047
	kcPause       uint16 = 0x0048
	kcInsert      uint16 = 0x0049
	kcHome        uint16 = 0x004A
	kcPageUp      uint16 = 0x004B
	kcDelete      uint16 = 0x004C
	kcEnd         uint16 = 0x004D
	kcPageDown    uint16 = 0x004E
	kcRight       uint16 = 0x004F
	kcLeft        uint16 = 0x0050
	kcDown        uint16 = 0x0051
	kcUp          uint16 = 0x0052
)

const (
	kcNumLock uint16 = 0x0053
	kcKPSlash uint16 = 0x0054
	kcKPStar  uint16 = 0x0055
	kcKPMinus uint16 = 0x0056
	kcKPPlus  uint16 = 0x0057
	kcKPEnter uint16 = 0x0058
	kcKP1     uint16 = 0x0059
	kcKP2     uint16 = 0x005A
	kcKP3     uint16 = 0x005B
	kcKP4     uint16 = 0x005C
	kcKP5     uint16 = 0x005D
	kcKP6     uint16 = 0x005E
	kcKP7     uint16 = 0x005F
	kcKP8     uint16 = 0x0060
	kcKP9     uint16 = 0x0061
	kcKP0     uint16 = 0x0062
	kcKPDot   uint16 = 0x0063
	kcNonUSBS uint16 = 0x0064
	kcApp     uint16 = 0x0065
	kcKBPower uint16 = 0x0066
	kcKPEqual uint16 = 0x0067
)

const (
	kcF13 uint16 = 0x0068 + iota
	kcF14
	kcF15
	kcF16
	kcF17
	kcF18
	kcF19
	kcF20
	kcF21
	kcF22
	kcF23
	kcF24
)

// Locking & alt-function keys.
const (
	kcLockingCapsLock   uint16 = 0x0082
	kcLockingNumLock    uint16 = 0x0083
	kcLockingScrollLock uint16 = 0x0084
	kcKPComma           uint16 = 0x0085
	kcKPEqualAS400      uint16 = 0x0086
)

const (
	kcInt1 uint16 = 0x0087 + iota
	kcInt2
	kcInt3
	kcInt4
	kcInt5
	kcInt6
	kcInt7
	kcInt8
	kcInt9
)

const (
	kcLang1 uint16 = 0x0090 + iota
	kcLang2
	kcLang3
	kcLang4
	kcLang5
	kcLang6
	kcLang7
	kcLang8
	kcLang9
)

const (
	kcAltErase   uint16 = 0x0099
	kcSysReq     uint16 = 0x009A
	kcCancel     uint16 = 0x009B
	kcClear      uint16 = 0x009C
	kcPrior      uint16 = 0x009D
	kcReturn     uint16 = 0x009E
	kcSeparator  uint16 = 0x009F
	kcOut        uint16 = 0x00A0
	kcOper       uint16 = 0x00A1
	kcClearAgain uint16 = 0x00A2
	kcCrSel      uint16 = 0x00A3
	kcExSel      uint16 = 0x00A4
)

// System power keys.
const (
	kcSystemPower uint16 = 0x00A5
	kcSystemSleep uint16 = 0x00A6
	kcSystemWake  uint16 = 0x00A7
)

// Audio / media keys.
const (
	kcAudioMute    uint16 = 0x00A8
	kcAudioVolUp   uint16 = 0x00A9
	kcAudioVolDown uint16 = 0x00AA
	kcMediaNext    uint16 = 0x00AB
	kcMediaPrev    uint16 = 0x00AC
	kcMediaStop    uint16 = 0x00AD
	kcMediaPlay    uint16 = 0x00AE
	kcMediaSelect  uint16 = 0x00AF
	kcMediaEject   uint16 = 0x00B0
)

// Web / launcher / system shortcut keys.
const (
	kcMail             uint16 = 0x00B1
	kcCalculator       uint16 = 0x00B2
	kcMyComputer       uint16 = 0x00B3
	kcWWWSearch        uint16 = 0x00B4
	kcWWWHome          uint16 = 0x00B5
	kcWWWBack          uint16 = 0x00B6
	kcWWWForward       uint16 = 0x00B7
	kcWWWStop          uint16 = 0x00B8
	kcWWWRefresh       uint16 = 0x00B9
	kcWWWFavorite      uint16 = 0x00BA
	kcMediaFastForward uint16 = 0x00BB
	kcMediaRewind      uint16 = 0x00BC
	kcBrightnessUp     uint16 = 0x00BD
	kcBrightnessDown   uint16 = 0x00BE
	kcControlPanel     uint16 = 0x00BF
	kcAssistant        uint16 = 0x00C0
	kcMissionControl   uint16 = 0x00C1
	kcLaunchpad        uint16 = 0x00C2
)

// Mouse keys.
const (
	kcMouseUp         uint16 = 0x00CD
	kcMouseDown       uint16 = 0x00CE
	kcMouseLeft       uint16 = 0x00CF
	kcMouseRight      uint16 = 0x00D0
	kcMouseBtn1       uint16 = 0x00D1
	kcMouseBtn2       uint16 = 0x00D2
	kcMouseBtn3       uint16 = 0x00D3
	kcMouseBtn4       uint16 = 0x00D4
	kcMouseBtn5       uint16 = 0x00D5
	kcMouseBtn6       uint16 = 0x00D6
	kcMouseBtn7       uint16 = 0x00D7
	kcMouseBtn8       uint16 = 0x00D8
	kcMouseWheelUp    uint16 = 0x00D9
	kcMouseWheelDown  uint16 = 0x00DA
	kcMouseWheelLeft  uint16 = 0x00DB
	kcMouseWheelRight uint16 = 0x00DC
	kcMouseAccel0     uint16 = 0x00DD
	kcMouseAccel1     uint16 = 0x00DE
	kcMouseAccel2     uint16 = 0x00DF
)

// basicKeycodes maps each non-composite QMK keycode to the short cap label
// rendered by Label(). Entries follow Remap's keycode picker text where
// possible, with multi-word labels split on spaces ("Caps\nLock" etc.) so
// the auto-shrinking renderer can stack tokens vertically.
var basicKeycodes = map[uint16]string{
	kcA: "A", kcB: "B", kcC: "C", kcD: "D",
	kcE: "E", kcF: "F", kcG: "G", kcH: "H",
	kcI: "I", kcJ: "J", kcK: "K", kcL: "L",
	kcM: "M", kcN: "N", kcO: "O", kcP: "P",
	kcQ: "Q", kcR: "R", kcS: "S", kcT: "T",
	kcU: "U", kcV: "V", kcW: "W", kcX: "X",
	kcY: "Y", kcZ: "Z",

	kc1: "1", kc2: "2", kc3: "3", kc4: "4", kc5: "5",
	kc6: "6", kc7: "7", kc8: "8", kc9: "9", kc0: "0",

	kcEnter: "Enter", kcEscape: "Esc", kcBackspace: "BS", kcTab: "Tab",
	kcSpace: "Space", kcMinus: "-", kcEqual: "=", kcLeftBracket: "[",
	kcRightBracket: "]", kcBackslash: "\\", kcNonUSHash: "#", kcSemicolon: ":\n;",
	kcQuote: "\"\n'", kcGrave: "`", kcComma: "<\n,", kcDot: ">\n.",
	kcSlash: "?\n/", kcCapsLock: "Caps\nLock",

	kcF1: "F1", kcF2: "F2", kcF3: "F3", kcF4: "F4",
	kcF5: "F5", kcF6: "F6", kcF7: "F7", kcF8: "F8",
	kcF9: "F9", kcF10: "F10", kcF11: "F11", kcF12: "F12",

	kcPrintScreen: "PrtSc", kcScrollLock: "ScrLk", kcPause: "Pause",
	kcInsert: "Ins", kcHome: "Home", kcPageUp: "PgUp",
	kcDelete: "Del", kcEnd: "End", kcPageDown: "PgDn",
	kcRight: "→", kcLeft: "←", kcDown: "↓", kcUp: "↑",

	kcNumLock: "NumLk", kcKPSlash: "KP/", kcKPStar: "KP*",
	kcKPMinus: "KP-", kcKPPlus: "KP+", kcKPEnter: "KPEnt",
	kcKP1: "KP1", kcKP2: "KP2", kcKP3: "KP3",
	kcKP4: "KP4", kcKP5: "KP5", kcKP6: "KP6",
	kcKP7: "KP7", kcKP8: "KP8", kcKP9: "KP9",
	kcKP0: "KP0", kcKPDot: "KP.",

	kcNonUSBS: "NUBS", kcApp: "App", kcKBPower: "Power", kcKPEqual: "KP=",

	kcF13: "F13", kcF14: "F14", kcF15: "F15", kcF16: "F16",
	kcF17: "F17", kcF18: "F18", kcF19: "F19", kcF20: "F20",
	kcF21: "F21", kcF22: "F22", kcF23: "F23", kcF24: "F24",

	kcLockingCapsLock:   "Locking\nCaps\nLock",
	kcLockingNumLock:    "Locking\nNum\nLock",
	kcLockingScrollLock: "Locking\nScroll\nLock",
	kcKPComma:           "KP,",
	kcKPEqualAS400:      "Num\n=\nAS400",

	kcInt1: "INT1", kcInt2: "INT2", kcInt3: "INT3",
	kcInt4: "INT4", kcInt5: "INT5", kcInt6: "INT6",
	kcInt7: "INT7", kcInt8: "INT8", kcInt9: "INT9",
	kcLang1: "LANG1", kcLang2: "LANG2", kcLang3: "LANG3",
	kcLang4: "LANG4", kcLang5: "LANG5", kcLang6: "LANG6",
	kcLang7: "LANG7", kcLang8: "LANG8", kcLang9: "LANG9",

	kcAltErase: "Alt\nErase", kcSysReq: "SysReq", kcCancel: "Cancel",
	kcClear: "Clear", kcPrior: "Prior", kcReturn: "Return",
	kcSeparator: "Separator",
	kcOut:       "Out", kcOper: "Oper",
	kcClearAgain: "Clear/\nAgain", kcCrSel: "CrSel/\nProps", kcExSel: "ExSel",

	kcSystemPower: "System\nPower\nDown", kcSystemSleep: "Sleep", kcSystemWake: "Wake",

	kcAudioMute:    "Audio\nMute",
	kcAudioVolUp:   "Audio\nVol +",
	kcAudioVolDown: "Audio\nVol -",
	kcMediaNext:    "Next",
	kcMediaPrev:    "Previous",
	kcMediaStop:    "Media\nStop",
	kcMediaPlay:    "Play",
	kcMediaSelect:  "Select",
	kcMediaEject:   "Eject",

	kcMail:             "Mail",
	kcCalculator:       "Calculator",
	kcMyComputer:       "My\nComputer",
	kcWWWSearch:        "WWW\nSearch",
	kcWWWHome:          "WWW\nHome",
	kcWWWBack:          "WWW\nBack",
	kcWWWForward:       "WWW\nForward",
	kcWWWStop:          "WWW\nStop",
	kcWWWRefresh:       "WWW\nRefresh",
	kcWWWFavorite:      "WWW\nFavorite",
	kcMediaFastForward: "Fast\nForward",
	kcMediaRewind:      "Rewind",
	kcBrightnessUp:     "Screen +",
	kcBrightnessDown:   "Screen -",
	kcControlPanel:     "Open\nControl\nPanel",
	kcAssistant:        "Assistant",
	kcMissionControl:   "Mission\nControl",
	kcLaunchpad:        "Launchpad",

	kcMouseUp: "Mouse\n↑", kcMouseDown: "Mouse\n↓",
	kcMouseLeft: "Mouse\n←", kcMouseRight: "Mouse\n→",
	kcMouseBtn1: "Mouse\nBtn1", kcMouseBtn2: "Mouse\nBtn2",
	kcMouseBtn3: "Mouse\nBtn3", kcMouseBtn4: "Mouse\nBtn4",
	kcMouseBtn5: "Mouse\nBtn5", kcMouseBtn6: "Mouse\nBtn6",
	kcMouseBtn7: "Mouse\nBtn7", kcMouseBtn8: "Mouse\nBtn8",
	kcMouseWheelUp: "Mouse\nWh ↑", kcMouseWheelDown: "Mouse\nWh ↓",
	kcMouseWheelLeft: "Mouse\nWh ←", kcMouseWheelRight: "Mouse\nWh →",
	kcMouseAccel0: "Mouse\nAcc0", kcMouseAccel1: "Mouse\nAcc1", kcMouseAccel2: "Mouse\nAcc2",
}
