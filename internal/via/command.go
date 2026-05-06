// Package via implements a strict read-only client for the VIA Raw HID
// protocol used by QMK/VIA-compatible firmwares.
//
// The client purposely cannot perform any keymap-mutation command. The
// command-byte constants for write operations (e.g. dynamic_keymap_set_keycode,
// reset_eeprom, jump_to_bootloader) are intentionally NOT declared here, so the
// codebase has no compile-time path that even constructs such a request. The
// runtime guard in writeReport additionally panics if a non-allowed CommandID
// somehow reaches it.
package via

// CommandID is the first byte of a VIA Raw HID payload. The full protocol
// defines many command IDs, but this read-only viewer only needs to read.
type CommandID uint8

const (
	// CmdProtocolVersion → reports the firmware's VIA protocol version.
	CmdProtocolVersion CommandID = 0x01

	// CmdGetKeycode → returns one keycode by (layer, row, col).
	// Mutating counterpart 0x05 (dynamic_keymap_set_keycode) is INTENTIONALLY
	// not declared here.
	CmdGetKeycode CommandID = 0x04

	// CmdGetLayerCount → returns the firmware's compiled-in layer count.
	CmdGetLayerCount CommandID = 0x11

	// CmdGetBuffer → bulk-read raw keymap memory (layers*rows*cols*2 bytes).
	// Mutating counterpart 0x13 (dynamic_keymap_set_buffer) is INTENTIONALLY
	// not declared here.
	CmdGetBuffer CommandID = 0x12
)

// allowedCommands is the runtime whitelist consulted by writeReport. Any
// other CommandID value triggers a panic before bytes touch the device. The
// set is exposed via AllowedCommands() so tests can assert the whitelist
// matches the read-only API surface.
var allowedCommands = map[CommandID]struct{}{
	CmdProtocolVersion: {},
	CmdGetKeycode:      {},
	CmdGetLayerCount:   {},
	CmdGetBuffer:       {},
}

// IsAllowed reports whether the given CommandID is on the read-only whitelist.
func IsAllowed(id CommandID) bool {
	_, ok := allowedCommands[id]
	return ok
}

// AllowedCommands returns a fresh slice copy of the whitelist (sorted is not
// guaranteed). Tests use this to compare against the public method set.
func AllowedCommands() []CommandID {
	out := make([]CommandID, 0, len(allowedCommands))
	for id := range allowedCommands {
		out = append(out, id)
	}
	return out
}
