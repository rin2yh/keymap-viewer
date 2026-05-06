// Package keymap parses VIA v3 keyboard definitions and renders QMK keycodes
// as human-readable labels for the read-only viewer.
package keymap

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

//go:embed crkbd.json
var embeddedCrkbdJSON []byte

// Matrix describes the keyboard's switch matrix dimensions.
type Matrix struct {
	Rows int `json:"rows"`
	Cols int `json:"cols"`
}

// Key is a single physical key in the rendered layout.
//
// X, Y are in unit cells (1.0 = one keycap). W, H default to 1. Rotation is in
// degrees, applied around (RotationOriginX, RotationOriginY). Decal keys are
// visual-only filler in VIA definitions and are skipped during parsing.
type Key struct {
	Row, Col        int
	X, Y            float64
	W, H            float64
	Rotation        float64
	RotationOriginX float64
	RotationOriginY float64
}

// Definition is the parsed view of a VIA v3 keyboard JSON definition.
type Definition struct {
	Name      string
	VendorID  uint16
	ProductID uint16
	Matrix    Matrix
	Keys      []Key
}

type rawDefinition struct {
	Name      string          `json:"name"`
	VendorID  string          `json:"vendorId"`
	ProductID string          `json:"productId"`
	Matrix    Matrix          `json:"matrix"`
	Layouts   rawLayouts      `json:"layouts"`
	Keymap    json.RawMessage `json:"-"`
}

type rawLayouts struct {
	Keymap []json.RawMessage `json:"keymap"`
}

// LoadEmbeddedDefinition parses the bundled crkbd.json shipped with the binary.
func LoadEmbeddedDefinition() (*Definition, error) {
	return ParseDefinition(embeddedCrkbdJSON)
}

// ParseDefinition parses raw VIA v3 JSON bytes.
func ParseDefinition(data []byte) (*Definition, error) {
	var raw rawDefinition
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("keymap: decode definition: %w", err)
	}
	if raw.Matrix.Rows <= 0 || raw.Matrix.Cols <= 0 {
		return nil, fmt.Errorf("keymap: invalid matrix dims rows=%d cols=%d", raw.Matrix.Rows, raw.Matrix.Cols)
	}
	vid, err := parseHexU16(raw.VendorID)
	if err != nil {
		return nil, fmt.Errorf("keymap: vendorId: %w", err)
	}
	pid, err := parseHexU16(raw.ProductID)
	if err != nil {
		return nil, fmt.Errorf("keymap: productId: %w", err)
	}
	keys, err := flattenKeymap(raw.Layouts.Keymap)
	if err != nil {
		return nil, err
	}
	return &Definition{
		Name:      raw.Name,
		VendorID:  vid,
		ProductID: pid,
		Matrix:    raw.Matrix,
		Keys:      keys,
	}, nil
}

func parseHexU16(s string) (uint16, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}
	s = strings.TrimPrefix(s, "0x")
	s = strings.TrimPrefix(s, "0X")
	v, err := strconv.ParseUint(s, 16, 16)
	if err != nil {
		return 0, err
	}
	return uint16(v), nil
}

// kleCursor is the running KLE deserialization state mutated as tokens are
// decoded.
type kleCursor struct {
	x, y     float64
	rotX     float64
	rotY     float64
	rotAngle float64
	w, h     float64
	decal    bool
}

// flattenKeymap walks the VIA v3 KLE-style array and emits one Key per
// "row,col" string, mirroring kle-serial's deserialization order:
//
//   - Attributes are processed in the JSON order they appear (so the spec's
//     `{r, rx, ry, y, x}` group works: r/rx/ry first reset x,y to (rx,ry),
//     then y/x apply as deltas relative to that origin).
//   - At end of each row, y++ and x is reset to the active rotation_x.
//   - Setting rx or ry resets BOTH x and y to (rotation_x, rotation_y).
//   - Setting r alone (rotation angle) does NOT touch x or y.
//
// See https://github.com/ijprest/kle-serial/blob/master/serial.ts for the
// canonical reference implementation.
func flattenKeymap(rows []json.RawMessage) ([]Key, error) {
	var out []Key
	c := kleCursor{w: 1, h: 1}

	for ri, rowRaw := range rows {
		var row []json.RawMessage
		if err := json.Unmarshal(rowRaw, &row); err != nil {
			return nil, fmt.Errorf("keymap: row %d: %w", ri, err)
		}

		for _, item := range row {
			trimmed := strings.TrimSpace(string(item))
			if len(trimmed) == 0 {
				continue
			}
			switch trimmed[0] {
			case '{':
				if err := applyAttrs(item, &c); err != nil {
					return nil, fmt.Errorf("keymap: row %d attr: %w", ri, err)
				}
			case '"':
				var s string
				if err := json.Unmarshal(item, &s); err != nil {
					return nil, fmt.Errorf("keymap: row %d label: %w", ri, err)
				}
				if !c.decal {
					rrow, rcol, err := parseRowCol(s)
					if err != nil {
						return nil, fmt.Errorf("keymap: row %d: %w", ri, err)
					}
					out = append(out, Key{
						Row:             rrow,
						Col:             rcol,
						X:               c.x,
						Y:               c.y,
						W:               c.w,
						H:               c.h,
						Rotation:        c.rotAngle,
						RotationOriginX: c.rotX,
						RotationOriginY: c.rotY,
					})
				}
				c.x += c.w
				c.w, c.h = 1, 1
				c.decal = false
			default:
				return nil, fmt.Errorf("keymap: row %d: unexpected token %q", ri, trimmed)
			}
		}

		// End of the outer row: kle-serial advances y by 1 and resets x to
		// the current rotation origin's x (0 when no rotation block is
		// active).
		c.y += 1
		c.x = c.rotX
	}

	return out, nil
}

// applyAttrs decodes a single KLE attribute object preserving the JSON key
// order — needed so `{r, rx, ry, y, x}` sets rotation first, snaps x/y to
// (rx, ry), then applies y/x as relative deltas.
func applyAttrs(raw []byte, c *kleCursor) error {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	t, err := dec.Token()
	if err != nil {
		return err
	}
	if d, ok := t.(json.Delim); !ok || d != '{' {
		return fmt.Errorf("expected '{' got %v", t)
	}
	for dec.More() {
		keyTok, err := dec.Token()
		if err != nil {
			return err
		}
		key, ok := keyTok.(string)
		if !ok {
			return fmt.Errorf("attribute key not string: %v", keyTok)
		}
		valTok, err := dec.Token()
		if err != nil {
			return err
		}
		switch key {
		case "r":
			if f, ok := numFloat(valTok); ok {
				c.rotAngle = f
			}
		case "rx":
			if f, ok := numFloat(valTok); ok {
				c.rotX = f
				c.x = c.rotX
				c.y = c.rotY
			}
		case "ry":
			if f, ok := numFloat(valTok); ok {
				c.rotY = f
				c.x = c.rotX
				c.y = c.rotY
			}
		case "x":
			if f, ok := numFloat(valTok); ok {
				c.x += f
			}
		case "y":
			if f, ok := numFloat(valTok); ok {
				c.y += f
			}
		case "w":
			if f, ok := numFloat(valTok); ok {
				c.w = f
			}
		case "h":
			if f, ok := numFloat(valTok); ok {
				c.h = f
			}
		case "d":
			if b, ok := valTok.(bool); ok {
				c.decal = b
			}
		default:
			// Skip unknown keys (incl. arrays/objects we don't model).
			if d, ok := valTok.(json.Delim); ok && (d == '[' || d == '{') {
				if err := skipNested(dec); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func numFloat(t json.Token) (float64, bool) {
	if v, ok := t.(json.Number); ok {
		if f, err := v.Float64(); err == nil {
			return f, true
		}
	}
	return 0, false
}

// skipNested drains tokens until the depth (already 1 from the caller's
// opening delimiter) returns to 0. The caller already consumed the opening
// '[' or '{', so this matches the closing ']' or '}'.
func skipNested(dec *json.Decoder) error {
	depth := 1
	for depth > 0 {
		t, err := dec.Token()
		if err != nil {
			return err
		}
		if d, ok := t.(json.Delim); ok {
			switch d {
			case '[', '{':
				depth++
			case ']', '}':
				depth--
			}
		}
	}
	return nil
}

func parseRowCol(s string) (int, int, error) {
	parts := strings.Split(s, ",")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("expected \"row,col\" got %q", s)
	}
	r, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return 0, 0, fmt.Errorf("row in %q: %w", s, err)
	}
	c, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return 0, 0, fmt.Errorf("col in %q: %w", s, err)
	}
	return r, c, nil
}
