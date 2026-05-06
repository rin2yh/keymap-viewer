package via

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// VIA Raw HID transport constants.
//
// VIA-compatible QMK firmwares expose a 32-byte payload in/out, prefixed with
// the standard HID Report ID byte 0x00 on platforms that require it (macOS,
// Windows). The first payload byte is the CommandID; the remaining 31 bytes
// hold command-specific arguments and zero-padding.
//
// See https://www.caniusevia.com/docs/specification.
const (
	// PayloadSize is the size of the VIA payload itself, excluding the leading
	// HID report ID byte.
	PayloadSize = 32

	// ReportSize is the size written to / read from the HID device, including
	// the leading report ID byte.
	ReportSize = PayloadSize + 1

	// VIAUsagePage and VIAUsage uniquely identify the VIA Raw HID interface
	// on devices that expose multiple HID interfaces.
	VIAUsagePage uint16 = 0xFF60
	VIAUsage     uint16 = 0x0061
)

// errShortRead indicates that the device returned fewer bytes than expected.
var errShortRead = errors.New("via: short read from device")

// buildRequest assembles a 33-byte HID OUT report for the given command. The
// returned slice's first byte is the report ID (0x00); the second byte is the
// CommandID echoed back by the firmware in the response. payload is the 0..30
// command-specific argument bytes that follow CommandID.
func buildRequest(id CommandID, payload []byte) ([]byte, error) {
	if len(payload) > PayloadSize-1 {
		return nil, fmt.Errorf("via: payload too large: %d > %d", len(payload), PayloadSize-1)
	}
	buf := make([]byte, ReportSize)
	buf[0] = 0x00
	buf[1] = byte(id)
	copy(buf[2:], payload)
	return buf, nil
}

// validateResponse checks that the response echoes the requested CommandID.
// The HID report ID byte is NOT present on the read side on macOS/Windows
// hidapi, so resp[0] is the CommandID echo.
func validateResponse(want CommandID, resp []byte) error {
	if len(resp) < 1 {
		return errShortRead
	}
	if got := CommandID(resp[0]); got != want {
		return fmt.Errorf("via: command echo mismatch: want 0x%02X got 0x%02X", want, got)
	}
	return nil
}

// keycodeFromBytes decodes the big-endian 16-bit keycode used in
// dynamic_keymap_get_keycode and dynamic_keymap_get_buffer responses.
func keycodeFromBytes(b []byte) uint16 {
	return binary.BigEndian.Uint16(b)
}
