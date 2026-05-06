package via

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	hid "github.com/sstallion/go-hid"

	"github.com/rin2yh/keymap-viewer/internal/keymap"
)

func isPermissionError(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "not permitted") ||
		strings.Contains(s, "0xE00002E2") ||
		strings.Contains(s, "Operation not permitted")
}

// CrkbdVendorID and CrkbdProductID are the VID/PID for Corne v4 Chocolate.
//
// foostan/corne v4 firmware reports product_id 0x0004 over USB. (The VIA
// definition file at the-via/keyboards/master/v3/crkbd/crkbd.json declares
// "productId": "0x0001", but that field is matched by VIA Web only against
// VIA-side metadata, not against USB descriptors — the wire-level product id
// the firmware itself advertises is 0x0004.)
const (
	CrkbdVendorID  uint16 = 0x4653
	CrkbdProductID uint16 = 0x0004
)

// defaultReadTimeout is the per-request HID read deadline. Real devices reply
// in well under a millisecond; we leave generous headroom for USB scheduling.
var defaultReadTimeout = 200 * time.Millisecond

// rawDevice is the subset of *hid.Device used by ReadOnlyClient. Tests
// substitute a fake to verify request bytes without a physical device.
type rawDevice interface {
	Write(p []byte) (int, error)
	ReadWithTimeout(p []byte, timeout time.Duration) (int, error)
	Close() error
}

// RawDevice is the public alias of the HID transport interface used by
// ReadOnlyClient. External test packages implement it to inject a fake
// device without spinning up a real HID stack.
type RawDevice = rawDevice

// Opener returns an opened ReadOnlyClient. Production code uses Open;
// tests pass a closure that builds a client around a fake transport.
type Opener func() (*ReadOnlyClient, error)

// NewFromDevice constructs a ReadOnlyClient around a caller-supplied
// transport. Production code should use Open; this constructor exists for
// E2E-style tests that need to drive the client with a programmable fake.
func NewFromDevice(dev RawDevice) *ReadOnlyClient {
	return &ReadOnlyClient{
		dev:     dev,
		readBuf: make([]byte, PayloadSize),
		timeout: defaultReadTimeout,
	}
}

// ReadOnlyClient is the public API for talking to a VIA-compatible device.
//
// By construction, ReadOnlyClient exposes ONLY get-style methods:
//
//   - ProtocolVersion
//   - LayerCount
//   - Keycode
//   - KeymapBuffer
//   - Close
//
// There is no setter method on this type, and the package's command
// whitelist (see allowedCommands) panics if any other command ID is sent.
type ReadOnlyClient struct {
	dev     rawDevice
	mu      sync.Mutex
	readBuf []byte
	timeout time.Duration
	devInfo *hid.DeviceInfo

	cachedLayerCount uint8
}

// errEnumerateStop is a sentinel returned from the hid.Enumerate callback to
// short-circuit iteration once the VIA interface is found.
var errEnumerateStop = errors.New("via: enumerate stop")

// Open enumerates HID devices for the Corne VID/PID, picks the one whose
// usage_page+usage match the VIA Raw HID interface, and opens it.
//
// On macOS and Windows hidapi enumerates each HID interface separately, so a
// VID/PID pair that exposes both a keyboard interface (usage_page 0x0001) and
// the VIA interface (usage_page 0xFF60) requires the path-level disambiguation
// performed here.
func Open() (*ReadOnlyClient, error) {
	if err := hid.Init(); err != nil {
		return nil, fmt.Errorf("via: hid.Init: %w", err)
	}
	var target *hid.DeviceInfo
	err := hid.Enumerate(CrkbdVendorID, CrkbdProductID, func(info *hid.DeviceInfo) error {
		if info.UsagePage == VIAUsagePage && info.Usage == VIAUsage {
			cp := *info
			target = &cp
			return errEnumerateStop
		}
		return nil
	})
	if err != nil && !errors.Is(err, errEnumerateStop) {
		return nil, fmt.Errorf("via: hid.Enumerate: %w", err)
	}
	if target == nil {
		return nil, fmt.Errorf("via: no VIA Raw HID interface found for VID=0x%04X PID=0x%04X (usage_page=0x%04X, usage=0x%04X). Add this terminal/IDE to System Settings → Privacy → Input Monitoring on macOS",
			CrkbdVendorID, CrkbdProductID, VIAUsagePage, VIAUsage)
	}
	dev, err := hid.OpenPath(target.Path)
	if err != nil {
		if isPermissionError(err) {
			return nil, fmt.Errorf(`via: hid.OpenPath %q: %w

This is a macOS Input Monitoring permission error. Even though the device is
visible to enumeration, opening the VIA Raw HID interface requires the
launching binary to be granted Input Monitoring access:

  1. System Settings → Privacy & Security → Input Monitoring
  2. Click '+' and add the *binary that is launching this program*
     (e.g. Terminal.app, iTerm2, your IDE, or the keymap-viewer binary itself).
  3. Fully QUIT and re-launch the launcher (Cmd-Q, not just close window).
  4. Re-run.`,
				target.Path, err)
		}
		return nil, fmt.Errorf("via: hid.OpenPath %q: %w", target.Path, err)
	}
	return &ReadOnlyClient{
		dev:     dev,
		readBuf: make([]byte, PayloadSize),
		timeout: defaultReadTimeout,
		devInfo: target,
	}, nil
}

// Close releases the underlying HID handle.
func (c *ReadOnlyClient) Close() error {
	if c == nil || c.dev == nil {
		return nil
	}
	err := c.dev.Close()
	c.dev = nil
	return err
}

// DeviceInfo returns metadata about the opened VIA interface, if any.
func (c *ReadOnlyClient) DeviceInfo() *hid.DeviceInfo {
	return c.devInfo
}

// ProtocolVersion reads the firmware's compiled-in VIA protocol version.
func (c *ReadOnlyClient) ProtocolVersion() (uint16, error) {
	resp, err := c.exchange(CmdProtocolVersion, nil)
	if err != nil {
		return 0, err
	}
	if len(resp) < 3 {
		return 0, errShortRead
	}
	return uint16(resp[1])<<8 | uint16(resp[2]), nil
}

// LayerCount reads the dynamic keymap's layer count. The result is cached on
// first success since layer count is firmware-constant per device session.
func (c *ReadOnlyClient) LayerCount() (uint8, error) {
	if n := c.cachedLayerCount; n != 0 {
		return n, nil
	}
	resp, err := c.exchange(CmdGetLayerCount, nil)
	if err != nil {
		return 0, err
	}
	if len(resp) < 2 {
		return 0, errShortRead
	}
	c.cachedLayerCount = resp[1]
	return resp[1], nil
}

// Keycode reads a single keycode at (layer, row, col).
func (c *ReadOnlyClient) Keycode(layer, row, col uint8) (uint16, error) {
	resp, err := c.exchange(CmdGetKeycode, []byte{layer, row, col})
	if err != nil {
		return 0, err
	}
	if len(resp) < 6 {
		return 0, errShortRead
	}
	// Response: [cmd, layer, row, col, kc_hi, kc_lo, ...]
	return keycodeFromBytes(resp[4:6]), nil
}

// KeymapBuffer reads `size` raw bytes of the dynamic keymap starting at
// `offset`. size is capped at 28 by the firmware (the response carries 28
// payload bytes after [cmd, off_hi, off_lo, size]).
func (c *ReadOnlyClient) KeymapBuffer(offset uint16, size uint8) ([]byte, error) {
	if size == 0 || size > 28 {
		return nil, fmt.Errorf("via: KeymapBuffer size out of range: %d", size)
	}
	payload := []byte{
		byte(offset >> 8),
		byte(offset & 0xFF),
		size,
	}
	resp, err := c.exchange(CmdGetBuffer, payload)
	if err != nil {
		return nil, err
	}
	if len(resp) < 4+int(size) {
		return nil, errShortRead
	}
	out := make([]byte, size)
	copy(out, resp[4:4+int(size)])
	return out, nil
}

// exchange performs one request/response round-trip with the firmware. The
// returned slice is a copy of the response payload (PayloadSize bytes), so
// callers may retain it across subsequent calls.
func (c *ReadOnlyClient) exchange(id CommandID, payload []byte) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.dev == nil {
		return nil, errors.New("via: client is closed")
	}

	if err := c.writeReport(id, payload); err != nil {
		return nil, err
	}

	n, err := c.dev.ReadWithTimeout(c.readBuf, c.timeout)
	if err != nil {
		return nil, fmt.Errorf("via: read: %w", err)
	}
	if n <= 0 {
		return nil, errShortRead
	}
	if err := validateResponse(id, c.readBuf[:n]); err != nil {
		return nil, err
	}
	out := make([]byte, n)
	copy(out, c.readBuf[:n])
	return out, nil
}

// writeReport is the only path that pushes bytes to the device. The whitelist
// check is intentionally a hard panic, not an error: a non-whitelisted command
// indicates a programming error in this package — the API surface is supposed
// to make such a call unreachable. Catching it here is defence in depth.
func (c *ReadOnlyClient) writeReport(id CommandID, payload []byte) error {
	if _, ok := allowedCommands[id]; !ok {
		panic(fmt.Sprintf("via: disallowed VIA command 0x%02X reached writeReport", byte(id)))
	}
	req, err := buildRequest(id, payload)
	if err != nil {
		return err
	}
	if _, err := c.dev.Write(req); err != nil {
		return fmt.Errorf("via: write: %w", err)
	}
	return nil
}

// ListMatchingDevices prints HID devices for diagnostics. It first tries to
// enumerate the Corne VID/PID; if no matches are found (typically a missing
// macOS Input Monitoring permission, or a different firmware VID/PID), it
// falls back to enumerating ALL HID devices on the system so the user can
// see whether anything is reachable at all and locate a usage_page=0xFF60
// entry by hand.
func ListMatchingDevices(w io.Writer) error {
	if err := hid.Init(); err != nil {
		return err
	}
	var matched int
	printDev := func(info *hid.DeviceInfo) error {
		fmt.Fprintf(w, "path=%s vid=0x%04X pid=0x%04X usage_page=0x%04X usage=0x%04X iface=%d mfr=%q product=%q\n",
			info.Path, info.VendorID, info.ProductID, info.UsagePage, info.Usage, info.InterfaceNbr, info.MfrStr, info.ProductStr)
		return nil
	}
	fmt.Fprintf(w, "[corne] enumerating VID=0x%04X PID=0x%04X ...\n", CrkbdVendorID, CrkbdProductID)
	if err := hid.Enumerate(CrkbdVendorID, CrkbdProductID, func(info *hid.DeviceInfo) error {
		matched++
		return printDev(info)
	}); err != nil {
		return err
	}
	if matched == 0 {
		fmt.Fprintln(w, "[corne] no matches. Falling back to all HID devices ...")
		fmt.Fprintln(w, "  - On macOS, an empty list usually means Input Monitoring permission is missing.")
		fmt.Fprintln(w, "  - Add the *binary itself* (not just the terminal) to Privacy → Input Monitoring,")
		fmt.Fprintln(w, "    fully quit and re-launch, then retry. Look for usage_page=0xFF60 in the list below.")
		fmt.Fprintln(w)
		var any int
		if err := hid.Enumerate(0, 0, func(info *hid.DeviceInfo) error {
			any++
			return printDev(info)
		}); err != nil {
			return err
		}
		if any == 0 {
			fmt.Fprintln(w, "[fallback] hid.Enumerate(0,0) returned ZERO devices — strong indicator of missing Input Monitoring permission.")
		}
	}
	return nil
}

// FetchSnapshot pulls every layer's keycodes for a (rows × cols) matrix using
// bulk KeymapBuffer reads, and returns a fully-populated *keymap.Snapshot.
func FetchSnapshot(c *ReadOnlyClient, rows, cols int) (*keymap.Snapshot, error) {
	layers, err := c.LayerCount()
	if err != nil {
		return nil, err
	}
	if layers == 0 {
		return nil, errors.New("via: device reports zero layers")
	}
	if rows <= 0 || cols <= 0 {
		return nil, fmt.Errorf("via: invalid matrix rows=%d cols=%d", rows, cols)
	}

	totalKeys := int(layers) * rows * cols
	totalBytes := totalKeys * 2
	raw := make([]byte, totalBytes)

	const chunk = 28
	for off := 0; off < totalBytes; off += chunk {
		size := chunk
		if off+size > totalBytes {
			size = totalBytes - off
		}
		buf, err := c.KeymapBuffer(uint16(off), uint8(size))
		if err != nil {
			return nil, fmt.Errorf("via: read buffer @0x%04X: %w", off, err)
		}
		copy(raw[off:off+size], buf)
	}

	snap := keymap.NewSnapshot(int(layers), rows, cols)
	idx := 0
	for l := range int(layers) {
		for r := range rows {
			for c := range cols {
				snap.Data[l][r][c] = keycodeFromBytes(raw[idx : idx+2])
				idx += 2
			}
		}
	}
	return snap, nil
}
