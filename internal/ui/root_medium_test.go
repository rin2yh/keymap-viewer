package ui

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/yuuki/keymap-viewer/internal/keymap"
	"github.com/yuuki/keymap-viewer/internal/via"
	"github.com/yuuki/keymap-viewer/internal/viatest"
)

// TestRoot_FetchesViaOpener verifies that the via.Opener passed to NewRoot
// is wired into the snapshot fetch path. The white-box assertions on
// pendingResult/snapshot are why this test lives in the ui package rather
// than e2e/.
func TestRoot_FetchesViaOpener(t *testing.T) {
	if testing.Short() {
		t.Skip("medium test; skipped under -short")
	}
	def, err := keymap.LoadEmbeddedDefinition()
	if err != nil {
		t.Fatalf("LoadEmbeddedDefinition: %v", err)
	}
	want := viatest.SampleSnapshot()

	var openCalls atomic.Int64
	root := NewRoot(def, func() (*via.ReadOnlyClient, error) {
		openCalls.Add(1)
		return via.NewFromDevice(viatest.NewFakeDevice(want)), nil
	})

	root.startFetch()

	deadline := time.Now().Add(2 * time.Second)
	var res *fetchResult
	for time.Now().Before(deadline) {
		if v := root.pendingResult.Swap(nil); v != nil {
			res = v
			break
		}
		time.Sleep(time.Millisecond)
	}
	if res == nil {
		t.Fatal("no pendingResult after 2s — opener was not called or goroutine never returned")
	}
	if res.err != nil {
		t.Fatalf("opener returned error: %v", res.err)
	}
	if got := openCalls.Load(); got != 1 {
		t.Errorf("opener invoked %d times, want 1", got)
	}
	got := res.snap
	if got == nil {
		t.Fatal("res.snap is nil")
	}
	if got.Layers != want.Layers {
		t.Errorf("Layers = %d, want %d", got.Layers, want.Layers)
	}
	if got.Rows != def.Matrix.Rows || got.Cols != def.Matrix.Cols {
		t.Errorf("dims = %dx%d, want %dx%d", got.Rows, got.Cols, def.Matrix.Rows, def.Matrix.Cols)
	}
	for l := 0; l < want.Layers; l++ {
		for r := 0; r < want.Rows; r++ {
			for c := 0; c < want.Cols; c++ {
				if got.Keycode(l, r, c) != want.Keycode(l, r, c) {
					t.Fatalf("keycode mismatch at %d,%d,%d: got 0x%04X want 0x%04X",
						l, r, c, got.Keycode(l, r, c), want.Keycode(l, r, c))
				}
			}
		}
	}
}
