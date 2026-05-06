package via

import (
	"reflect"
	"sort"
	"testing"
	"time"
)

func newClientForTest(d rawDevice) *ReadOnlyClient {
	return NewFromDevice(d)
}

// fakeRawDevice is an in-memory rawDevice used by tests. It records every
// Write and serves a canned response on each ReadWithTimeout. Neither
// operation ever returns an error.
type fakeRawDevice struct {
	writes   [][]byte
	response []byte
	closed   bool
}

func (f *fakeRawDevice) Write(p []byte) (int, error) {
	cp := make([]byte, len(p))
	copy(cp, p)
	f.writes = append(f.writes, cp)
	return len(p), nil
}

func (f *fakeRawDevice) ReadWithTimeout(p []byte, _ time.Duration) (int, error) {
	n := copy(p, f.response)
	return n, nil
}

func (f *fakeRawDevice) Close() error {
	f.closed = true
	return nil
}

func TestAllowedCommandsHasOnlyReads(t *testing.T) {
	got := AllowedCommands()
	if len(got) != 4 {
		t.Fatalf("AllowedCommands has %d entries, want 4: %v", len(got), got)
	}
	want := map[CommandID]bool{
		CmdProtocolVersion: true,
		CmdGetKeycode:      true,
		CmdGetLayerCount:   true,
		CmdGetBuffer:       true,
	}
	if len(want) != 4 {
		t.Fatalf("test expectation broken: want set has %d entries", len(want))
	}
	for _, id := range got {
		if !want[id] {
			t.Errorf("AllowedCommands contains disallowed id 0x%02X", byte(id))
		}
	}
	// Cross-check: the literal byte values are exactly the read-only ones.
	gotBytes := make([]int, len(got))
	for i, id := range got {
		gotBytes[i] = int(byte(id))
	}
	sort.Ints(gotBytes)
	wantBytes := []int{0x01, 0x04, 0x11, 0x12}
	if !reflect.DeepEqual(gotBytes, wantBytes) {
		t.Errorf("AllowedCommands bytes = %v, want %v", gotBytes, wantBytes)
	}
}

func TestWriteWhitelist_PanicsForDisallowed(t *testing.T) {
	for i := range 256 {
		id := CommandID(i)
		_, allowed := allowedCommands[id]

		fake := &fakeRawDevice{}
		client := newClientForTest(fake)

		func() {
			defer func() {
				r := recover()
				switch {
				case allowed && r != nil:
					t.Errorf("writeReport(0x%02X) panicked but is whitelisted: %v", i, r)
				case !allowed && r == nil:
					t.Errorf("writeReport(0x%02X) did not panic but is NOT whitelisted", i)
				}
			}()
			_ = client.writeReport(id, nil)
		}()
	}
}

func TestNoWriteSurface_PublicAPI(t *testing.T) {
	want := map[string]bool{
		"ProtocolVersion": true,
		"LayerCount":      true,
		"Keycode":         true,
		"KeymapBuffer":    true,
		"Close":           true,
		"DeviceInfo":      true,
	}

	typ := reflect.TypeFor[*ReadOnlyClient]()
	got := make(map[string]bool, typ.NumMethod())
	for m := range typ.Methods() {
		got[m.Name] = true
	}

	if typ.NumMethod() != len(want) {
		t.Errorf("(*ReadOnlyClient) has %d exported methods, want %d (got=%v want=%v)",
			typ.NumMethod(), len(want), keys(got), keys(want))
	}
	for name := range want {
		if !got[name] {
			t.Errorf("missing expected method %q on *ReadOnlyClient", name)
		}
	}
	for name := range got {
		if !want[name] {
			t.Errorf("unexpected method %q on *ReadOnlyClient (read-only API has been broken)", name)
		}
	}
}

func keys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func TestExchange_RoundTrip(t *testing.T) {
	// Canned response for CmdProtocolVersion: [cmd=0x01, 0x00, 0x0C, ...].
	// validateResponse reads resp[0] as the cmd echo (no leading report ID).
	resp := make([]byte, PayloadSize)
	resp[0] = byte(CmdProtocolVersion)
	resp[1] = 0x00
	resp[2] = 0x0C

	fake := &fakeRawDevice{response: resp}
	client := newClientForTest(fake)

	ver, err := client.ProtocolVersion()
	if err != nil {
		t.Fatalf("ProtocolVersion: %v", err)
	}
	if ver != 0x000C {
		t.Errorf("ProtocolVersion = 0x%04X, want 0x000C", ver)
	}

	if got := len(fake.writes); got != 1 {
		t.Fatalf("fake.writes len = %d, want 1", got)
	}
	req := fake.writes[0]
	if len(req) != ReportSize {
		t.Errorf("request size = %d, want %d", len(req), ReportSize)
	}
	if req[0] != 0x00 {
		t.Errorf("request[0] (report id) = 0x%02X, want 0x00", req[0])
	}
	if req[1] != byte(CmdProtocolVersion) {
		t.Errorf("request[1] (command) = 0x%02X, want 0x%02X", req[1], byte(CmdProtocolVersion))
	}
}
