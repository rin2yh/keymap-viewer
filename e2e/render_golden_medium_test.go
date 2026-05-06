// SPDX-License-Identifier: Apache-2.0

// Package e2e contains end-to-end tests that exercise the full read path
// from a fake VIA HID device through via.FetchSnapshot up to the data the
// UI's Keyboard widget would render per layer.
//
// The golden output captures position-and-label tuples in unit space
// (pre-pixel scaling) so the assertions are stable across host displays
// and font configurations.
package e2e

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/yuuki/keymap-viewer/internal/keymap"
	"github.com/yuuki/keymap-viewer/internal/via"
	"github.com/yuuki/keymap-viewer/internal/viatest"
)

var updateGolden = flag.Bool("update", false, "rewrite e2e/testdata/golden/*.txt")

// TestKeymapRender_Golden walks every layer of the fixture snapshot through
// the same path the GUI uses (FakeDevice → ReadOnlyClient → FetchSnapshot →
// Snapshot.Keycode → keymap.Label) and diffs the resulting per-key output
// against e2e/testdata/golden/layer_<n>.txt.
//
// The `keymap.Definition` parsed from the embedded crkbd.json is iterated
// in declaration order; each line records the key's logical position
// (unit space) and rendered label, matching what Keyboard.Build would
// emit at runtime.
func TestKeymapRender_Golden(t *testing.T) {
	if testing.Short() {
		t.Skip("medium test; skipped under -short")
	}
	def, err := keymap.LoadEmbeddedDefinition()
	if err != nil {
		t.Fatalf("LoadEmbeddedDefinition: %v", err)
	}

	want := viatest.SampleSnapshot()

	client := via.NewFromDevice(viatest.NewFakeDevice(want))
	defer client.Close()

	got, err := via.FetchSnapshot(client, def.Matrix.Rows, def.Matrix.Cols)
	if err != nil {
		t.Fatalf("FetchSnapshot: %v", err)
	}
	if got.Layers != want.Layers || got.Rows != want.Rows || got.Cols != want.Cols {
		t.Fatalf("dims: got %dx%dx%d want %dx%dx%d",
			got.Layers, got.Rows, got.Cols,
			want.Layers, want.Rows, want.Cols)
	}

	for layer := 0; layer < got.Layers; layer++ {
		var buf bytes.Buffer
		for _, key := range def.Keys {
			label := keymap.Label(got.Keycode(layer, key.Row, key.Col))
			fmt.Fprintf(&buf,
				"(%d,%d) pos=(%g,%g) size=(%g,%g) rot=%g origin=(%g,%g) label=%q\n",
				key.Row, key.Col,
				key.X, key.Y,
				key.W, key.H,
				key.Rotation,
				key.RotationOriginX, key.RotationOriginY,
				label,
			)
		}

		path := filepath.Join("testdata", "golden", fmt.Sprintf("layer_%d.txt", layer))
		if *updateGolden {
			if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
				t.Fatalf("write %s: %v", path, err)
			}
			continue
		}
		expected, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s (run `go test ./e2e -update` to generate): %v", path, err)
		}
		if !bytes.Equal(buf.Bytes(), expected) {
			t.Errorf("layer %d: golden mismatch (run `go test ./e2e -update` to regenerate)\n"+
				"--- got ---\n%s\n--- want ---\n%s",
				layer, buf.String(), string(expected))
		}
	}
}
