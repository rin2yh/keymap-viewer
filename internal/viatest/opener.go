// SPDX-License-Identifier: Apache-2.0

package viatest

import (
	"github.com/yuuki/keymap-viewer/internal/keymap"
	"github.com/yuuki/keymap-viewer/internal/via"
)

// OpenerFromSnapshot returns a function compatible with ui.ClientOpener
// that, on each call, builds a fresh FakeDevice serving the given snapshot
// and wraps it in a *via.ReadOnlyClient.
//
// The return type is the bare function literal (not ui.ClientOpener) so
// this package does not need to depend on internal/ui — the value is
// directly assignable to ui.ClientOpener at the call site.
func OpenerFromSnapshot(snap *keymap.Snapshot) func() (*via.ReadOnlyClient, error) {
	return func() (*via.ReadOnlyClient, error) {
		return via.NewFromDevice(NewFakeDevice(snap)), nil
	}
}
