//go:build smoke

package via_test

import (
	"testing"

	"github.com/rin2yh/keymap-viewer/internal/keymap"
	"github.com/rin2yh/keymap-viewer/internal/via"
)

// TestRealHID_Smoke talks to a physically connected Corne v4 Chocolate.
// The smoke build tag keeps it out of every default test run; opt in with:
//
//	go test -tags smoke ./...
//
// via.Open failure still self-skips so the user gets a clear "no device"
// message rather than a hard failure.
func TestRealHID_Smoke(t *testing.T) {
	client, err := via.Open()
	if err != nil {
		t.Skipf("real device not reachable: %v", err)
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
