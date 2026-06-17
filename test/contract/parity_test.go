// Package contract holds the golden-contract harness: it diffs ms_home HOME
// responses against captured digital_bff responses to prove behavioral parity.
//
// Fixtures live in test/contract/fixtures as pairs:
//
//	<case>.golden.json  — captured from digital_bff (source of truth)
//	<case>.actual.json  — captured from ms_home for the same request
//
// The comparison is structural (JSON value equality; object key order ignored,
// array order significant). See test/contract/README.md for the capture workflow.
// With no fixtures present the test skips, so CI stays green until fixtures land.
package contract

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestHomeParity(t *testing.T) {
	goldens, _ := filepath.Glob("fixtures/*.golden.json")
	if len(goldens) == 0 {
		t.Skip("no fixtures in test/contract/fixtures — see README.md to capture them")
	}

	for _, golden := range goldens {
		name := strings.TrimSuffix(filepath.Base(golden), ".golden.json")
		t.Run(name, func(t *testing.T) {
			actual := strings.Replace(golden, ".golden.json", ".actual.json", 1)
			want := loadJSON(t, golden)
			got := loadJSON(t, actual)
			if !reflect.DeepEqual(want, got) {
				t.Errorf("parity mismatch for %q:\n  first difference at: %s", name, firstDiff("$", want, got))
			}
		})
	}
}

func loadJSON(t *testing.T, path string) any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	return v
}

// firstDiff returns a JSON-path-ish description of the first structural difference.
func firstDiff(path string, want, got any) string {
	if reflect.DeepEqual(want, got) {
		return ""
	}
	wm, wok := want.(map[string]any)
	gm, gok := got.(map[string]any)
	if wok && gok {
		for k, wv := range wm {
			gv, ok := gm[k]
			if !ok {
				return path + "." + k + " (missing in actual)"
			}
			if d := firstDiff(path+"."+k, wv, gv); d != "" {
				return d
			}
		}
		for k := range gm {
			if _, ok := wm[k]; !ok {
				return path + "." + k + " (unexpected in actual)"
			}
		}
	}
	return path
}
