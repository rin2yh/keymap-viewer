package keymap_test

import (
	"strings"
	"testing"

	"github.com/rin2yh/keymap-viewer/internal/keymap"
)

func TestParseDefinition_Crkbd(t *testing.T) {
	def, err := keymap.LoadEmbeddedDefinition()
	if err != nil {
		t.Fatalf("LoadEmbeddedDefinition: %v", err)
	}

	if def.Name != "Crkbd" {
		t.Errorf("Name = %q, want %q", def.Name, "Crkbd")
	}
	if def.VendorID != 0x4653 {
		t.Errorf("VendorID = 0x%04X, want 0x4653", def.VendorID)
	}
	if def.ProductID != 0x0001 {
		t.Errorf("ProductID = 0x%04X, want 0x0001", def.ProductID)
	}
	if def.Matrix.Rows != 8 {
		t.Errorf("Matrix.Rows = %d, want 8", def.Matrix.Rows)
	}
	if def.Matrix.Cols != 7 {
		t.Errorf("Matrix.Cols = %d, want 7", def.Matrix.Cols)
	}
	if got := len(def.Keys); got != 46 {
		t.Errorf("len(Keys) = %d, want 46", got)
	}

	seen := make(map[[2]int]bool, len(def.Keys))
	for i, k := range def.Keys {
		if k.Row < 0 || k.Row >= def.Matrix.Rows {
			t.Errorf("key[%d] Row = %d, out of range [0,%d)", i, k.Row, def.Matrix.Rows)
		}
		if k.Col < 0 || k.Col >= def.Matrix.Cols {
			t.Errorf("key[%d] Col = %d, out of range [0,%d)", i, k.Col, def.Matrix.Cols)
		}
		key := [2]int{k.Row, k.Col}
		if seen[key] {
			t.Errorf("duplicate (row,col) = (%d,%d) at index %d", k.Row, k.Col, i)
		}
		seen[key] = true
	}
}

func TestParseDefinition_Rejects(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		wantSub string
	}{
		{
			name:    "empty bytes",
			data:    []byte{},
			wantSub: "decode definition",
		},
		{
			name:    "malformed JSON",
			data:    []byte(`{"name": "x"`),
			wantSub: "decode definition",
		},
		{
			name:    "missing matrix dims",
			data:    []byte(`{"name":"x","vendorId":"0x0001","productId":"0x0001","matrix":{"rows":0,"cols":0},"layouts":{"keymap":[]}}`),
			wantSub: "invalid matrix dims",
		},
		{
			name:    "negative matrix rows",
			data:    []byte(`{"name":"x","vendorId":"0x0001","productId":"0x0001","matrix":{"rows":-1,"cols":2},"layouts":{"keymap":[]}}`),
			wantSub: "invalid matrix dims",
		},
		{
			name:    "bogus row,col strings",
			data:    []byte(`{"name":"x","vendorId":"0x0001","productId":"0x0001","matrix":{"rows":2,"cols":2},"layouts":{"keymap":[["abc,def"]]}}`),
			wantSub: "row in",
		},
		{
			name:    "missing comma in row,col",
			data:    []byte(`{"name":"x","vendorId":"0x0001","productId":"0x0001","matrix":{"rows":2,"cols":2},"layouts":{"keymap":[["00"]]}}`),
			wantSub: "expected \"row,col\"",
		},
		{
			name:    "bad vendorId hex",
			data:    []byte(`{"name":"x","vendorId":"zzz","productId":"0x0001","matrix":{"rows":1,"cols":1},"layouts":{"keymap":[]}}`),
			wantSub: "vendorId",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			def, err := keymap.ParseDefinition(tc.data)
			if err == nil {
				t.Fatalf("ParseDefinition succeeded, want error containing %q (def=%+v)", tc.wantSub, def)
			}
			if tc.wantSub != "" && !strings.Contains(err.Error(), tc.wantSub) {
				t.Errorf("error %q does not contain %q", err.Error(), tc.wantSub)
			}
		})
	}
}
