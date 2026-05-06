// SPDX-License-Identifier: Apache-2.0

package via_test

import (
	"testing"

	"github.com/yuuki/keymap-viewer/internal/keymap"
	"github.com/yuuki/keymap-viewer/internal/via"
)

// TestRealHID_Smoke is a manual smoke test that talks to a physically
// connected Corne v4 Chocolate. It is intentionally *not* gated by a
// build tag so that `go test ./...` always considers it; instead, the
// test self-skips when:
//
//   - `testing.Short()` is true (the CI configuration always passes
//     `-short`, so this guarantees CI never tries to talk to hardware), or
//   - via.Open() fails (no device attached, or no Input Monitoring grant).
//
// To run locally with a device connected:
//
//	go test ./internal/via -v -run TestRealHID_Smoke
func TestRealHID_Smoke(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping real-HID smoke in -short mode")
	}
	client, err := via.Open()
	if err != nil {
		t.Skipf("skipping: real device not reachable: %v", err)
	}
	defer client.Close()

	def, err := keymap.LoadEmbeddedDefinition()
	if err != nil {
		t.Fatalf("LoadEmbeddedDefinition: %v", err)
	}

	ver, err := client.ProtocolVersion()
	if err != nil {
		t.Fatalf("ProtocolVersion: %v", err)
	}
	if ver == 0 {
		t.Errorf("ProtocolVersion = 0; firmware should report a non-zero version")
	}
	t.Logf("VIA protocol version = 0x%04X", ver)

	layers, err := client.LayerCount()
	if err != nil {
		t.Fatalf("LayerCount: %v", err)
	}
	if layers == 0 {
		t.Fatal("LayerCount = 0; firmware should report at least one layer")
	}
	t.Logf("layer count = %d", layers)

	snap, err := via.FetchSnapshot(client, def.Matrix.Rows, def.Matrix.Cols)
	if err != nil {
		t.Fatalf("FetchSnapshot: %v", err)
	}
	if snap.Layers != int(layers) {
		t.Errorf("snapshot Layers = %d, want %d", snap.Layers, layers)
	}
	if snap.Rows != def.Matrix.Rows || snap.Cols != def.Matrix.Cols {
		t.Errorf("snapshot dims = %dx%d, want %dx%d",
			snap.Rows, snap.Cols, def.Matrix.Rows, def.Matrix.Cols)
	}

	// At least one keycode in layer 0 should be non-empty on a real
	// configured keyboard. This catches "device responded but is silent"
	// regressions (e.g. wrong endianness, off-by-one offset).
	var nonZero int
	for r := 0; r < snap.Rows; r++ {
		for c := 0; c < snap.Cols; c++ {
			if snap.Keycode(0, r, c) != 0 {
				nonZero++
			}
		}
	}
	if nonZero == 0 {
		t.Error("layer 0 snapshot is entirely zero; device read may have failed")
	}
}
