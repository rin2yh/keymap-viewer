// SPDX-License-Identifier: Apache-2.0

// Package ui hosts the read-only Remap viewer's guigui widget tree.
package ui

import (
	// Side-effect import: registers the bundled CJK font so labels containing
	// 日本語 (e.g. layer descriptions) render without missing glyphs.
	_ "github.com/guigui-gui/guigui/basicwidget/cjkfont"
)
