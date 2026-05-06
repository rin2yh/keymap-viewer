// Package viatest provides programmable in-memory fakes that satisfy the
// via.RawDevice transport, for use in E2E tests that need to drive the UI
// or the via.ReadOnlyClient without a physical HID device.
package viatest

import (
	"encoding/binary"
	"time"

	"github.com/yuuki/keymap-viewer/internal/keymap"
	"github.com/yuuki/keymap-viewer/internal/via"
)

// fakeProtocolVersion is what FakeDevice reports for CmdProtocolVersion.
// VIA spec versions are big-endian 16-bit; 0x000C corresponds to v0.0.12.
const fakeProtocolVersion uint16 = 0x000C

// FakeDevice implements via.RawDevice by replaying canned responses to the
// four read-only VIA commands.
type FakeDevice struct {
	snapshot  *keymap.Snapshot
	keymapBuf []byte

	// pending is the response queued for the next ReadWithTimeout, set on
	// each Write so the request/response pairing is preserved.
	pending []byte
}

// NewFakeDevice returns a FakeDevice that serves the given snapshot for
// CmdGetLayerCount / CmdGetKeycode / CmdGetBuffer.
func NewFakeDevice(snap *keymap.Snapshot) *FakeDevice {
	return &FakeDevice{
		snapshot:  snap,
		keymapBuf: flattenSnapshot(snap),
	}
}

func (f *FakeDevice) Write(p []byte) (int, error) {
	f.pending = f.respond(p)
	return len(p), nil
}

func (f *FakeDevice) ReadWithTimeout(p []byte, _ time.Duration) (int, error) {
	n := copy(p, f.pending)
	f.pending = nil
	return n, nil
}

func (f *FakeDevice) Close() error { return nil }

// respond builds the 32-byte VIA response payload for a 33-byte request
// frame: [report_id=0x00, cmd, args...].
func (f *FakeDevice) respond(req []byte) []byte {
	resp := make([]byte, via.PayloadSize)
	cmd := via.CommandID(req[1])
	resp[0] = byte(cmd)

	switch cmd {
	case via.CmdProtocolVersion:
		binary.BigEndian.PutUint16(resp[1:3], fakeProtocolVersion)

	case via.CmdGetLayerCount:
		if f.snapshot != nil {
			resp[1] = byte(f.snapshot.Layers)
		}

	case via.CmdGetKeycode:
		layer, row, col := int(req[2]), int(req[3]), int(req[4])
		resp[1], resp[2], resp[3] = byte(layer), byte(row), byte(col)
		kc := uint16(0)
		if f.snapshot != nil {
			kc = f.snapshot.Keycode(layer, row, col)
		}
		binary.BigEndian.PutUint16(resp[4:6], kc)

	case via.CmdGetBuffer:
		offset := int(binary.BigEndian.Uint16(req[2:4]))
		size := int(req[4])
		resp[1], resp[2], resp[3] = req[2], req[3], req[4]
		end := offset + size
		if end > len(f.keymapBuf) {
			end = len(f.keymapBuf)
		}
		if offset < end {
			copy(resp[4:4+(end-offset)], f.keymapBuf[offset:end])
		}
	}
	return resp
}

// flattenSnapshot serializes a snapshot to the [layer][row][col] big-endian
// byte order used by VIA's CmdGetBuffer.
func flattenSnapshot(snap *keymap.Snapshot) []byte {
	if snap == nil {
		return nil
	}
	buf := make([]byte, snap.Layers*snap.Rows*snap.Cols*2)
	idx := 0
	for l := 0; l < snap.Layers; l++ {
		for r := 0; r < snap.Rows; r++ {
			for c := 0; c < snap.Cols; c++ {
				binary.BigEndian.PutUint16(buf[idx:idx+2], snap.Data[l][r][c])
				idx += 2
			}
		}
	}
	return buf
}
