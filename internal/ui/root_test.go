// SPDX-License-Identifier: Apache-2.0

package ui

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/yuuki/keymap-viewer/internal/keymap"
	"github.com/yuuki/keymap-viewer/internal/via"
	"github.com/yuuki/keymap-viewer/internal/viatest"
)

// TestRoot_FetchesViaOpener verifies that Root.SetOpener wires a custom
// ClientOpener into the snapshot fetch path. The white-box assertions
// (private fields like pendingResult/snapshot) are why this test lives in
// the ui package rather than e2e/.
func TestRoot_FetchesViaOpener(t *testing.T) {
	def, err := keymap.LoadEmbeddedDefinition()
	if err != nil {
		t.Fatalf("LoadEmbeddedDefinition: %v", err)
	}
	want := viatest.SampleSnapshot()

	var openCalls atomic.Int64
	root := NewRoot(def)
	root.SetOpener(func() (*via.ReadOnlyClient, error) {
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
					t.Fatalf("keycode mismatch at (%d,%d,%d): got 0x%04X want 0x%04X",
						l, r, c, got.Keycode(l, r, c), want.Keycode(l, r, c))
				}
			}
		}
	}
}

// TestRoot_FallsBackToViaOpen confirms the default opener (when none is
// set via SetOpener) routes through via.Open. We cannot exercise the
// happy path without real hardware, so the assertion is that fetching
// without an opener surfaces the via.Open error in pendingResult — which
// proves the fallback was taken instead of a panic.
func TestRoot_FallsBackToViaOpen(t *testing.T) {
	def, err := keymap.LoadEmbeddedDefinition()
	if err != nil {
		t.Fatalf("LoadEmbeddedDefinition: %v", err)
	}
	root := NewRoot(def)
	root.startFetch()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if v := root.pendingResult.Swap(nil); v != nil {
			if v.err == nil {
				t.Skip("via.Open unexpectedly succeeded (real device attached?)")
			}
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatal("no pendingResult after 5s")
}
