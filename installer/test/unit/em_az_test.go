package unit_test

import (
	"reflect"
	"testing"

	"ydb-installer/internal/services/config"
)

func TestNormalizeAZLocations_Variants(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		want []string
	}{
		{
			name: "passes through valid list",
			in:   []string{"dc-1", "dc-2"},
			want: []string{"dc-1", "dc-2"},
		},
		{
			name: "trims whitespace",
			in:   []string{" dc-1 ", "dc-2\t"},
			want: []string{"dc-1", "dc-2"},
		},
		{
			name: "drops empty entries",
			in:   []string{"dc-1", "", "   ", "dc-3"},
			want: []string{"dc-1", "dc-3"},
		},
		{
			name: "nil falls back to defaults",
			in:   nil,
			want: []string{"dc-1", "dc-2", "dc-3"},
		},
		{
			name: "all-empty falls back to defaults",
			in:   []string{"", "  "},
			want: []string{"dc-1", "dc-2", "dc-3"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := config.NormalizeAZLocations(tc.in)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("NormalizeAZLocations(%v) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

func TestParseAZLocations_YAMLSequence(t *testing.T) {
	raw := []interface{}{"dc-1", "dc-2", "dc-3"}
	got := config.ParseAZLocations(raw)
	want := []string{"dc-1", "dc-2", "dc-3"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ParseAZLocations(%v) = %v, want %v", raw, got, want)
	}
}

func TestParseAZLocations_StringSlice(t *testing.T) {
	raw := []string{"dc-1", " dc-2 "}
	got := config.ParseAZLocations(raw)
	want := []string{"dc-1", "dc-2"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ParseAZLocations(%v) = %v, want %v", raw, got, want)
	}
}

// TestParseAZLocations_BracketStringUnsupported locks in the format cutover:
// the legacy scalar "[dc-1 dc-2 dc-3]" form is no longer parsed. Callers get
// nil, after which the writer re-emits a proper YAML sequence.
func TestParseAZLocations_BracketStringUnsupported(t *testing.T) {
	for _, raw := range []interface{}{
		"[dc-1 dc-2 dc-3]",
		"[]",
		"",
		"dc-1",
		42,
		nil,
	} {
		if got := config.ParseAZLocations(raw); got != nil {
			t.Errorf("ParseAZLocations(%#v) = %v, want nil", raw, got)
		}
	}
}

func TestParseAZLocations_SkipsNonStringItems(t *testing.T) {
	raw := []interface{}{"dc-1", 7, "", "dc-2"}
	got := config.ParseAZLocations(raw)
	want := []string{"dc-1", "dc-2"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ParseAZLocations(%v) = %v, want %v", raw, got, want)
	}
}

func TestDeriveAZFromDCs_Variants(t *testing.T) {
	tests := []struct {
		name          string
		dc1, dc2, dc3 []string
		want          []string
	}{
		{
			name: "all three DCs populated",
			dc1:  []string{"h1"},
			dc2:  []string{"h2"},
			dc3:  []string{"h3"},
			want: []string{"dc-1", "dc-2", "dc-3"},
		},
		{
			name: "only first DC",
			dc1:  []string{"h1", "h2"},
			want: []string{"dc-1"},
		},
		{
			name: "gap in the middle preserves order",
			dc1:  []string{"h1"},
			dc3:  []string{"h3"},
			want: []string{"dc-1", "dc-3"},
		},
		{
			name: "empty strings do not count",
			dc1:  []string{""},
			dc2:  []string{"h2"},
			want: []string{"dc-2"},
		},
		{
			name: "no hosts at all falls back to defaults",
			want: []string{"dc-1", "dc-2", "dc-3"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := config.DeriveAZFromDCs(tc.dc1, tc.dc2, tc.dc3)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("DeriveAZFromDCs = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestDefaultAZLocations_NotSharedReference(t *testing.T) {
	a := config.DefaultAZLocations()
	a[0] = "mutated"
	if b := config.DefaultAZLocations(); b[0] != "dc-1" {
		t.Errorf("DefaultAZLocations returned a shared slice: %v", b)
	}
}

// TestDefaultCPPresets_NodeShape verifies the preset defaults are emitted as a
// flow-style sequence of {id, cpu, memory} string mappings.
func TestDefaultCPPresets_NodeShape(t *testing.T) {
	node := config.DefaultCPPresets()
	if node == nil {
		t.Fatal("DefaultCPPresets returned nil")
	}
	if len(node.Content) != 3 {
		t.Fatalf("expected 3 presets, got %d", len(node.Content))
	}

	wantIDs := []string{"m1.medium", "m1.tiny", "m1.large"}
	for i, entry := range node.Content {
		if len(entry.Content) != 6 {
			t.Fatalf("preset[%d]: expected 3 key/value pairs, got %d nodes", i, len(entry.Content))
		}
		keys := []string{entry.Content[0].Value, entry.Content[2].Value, entry.Content[4].Value}
		if !reflect.DeepEqual(keys, []string{"id", "cpu", "memory"}) {
			t.Errorf("preset[%d] keys = %v, want [id cpu memory]", i, keys)
		}
		if entry.Content[1].Value != wantIDs[i] {
			t.Errorf("preset[%d].id = %q, want %q", i, entry.Content[1].Value, wantIDs[i])
		}
		// cpu/memory must be strings so ansible receives them as such.
		for _, valueIdx := range []int{3, 5} {
			if entry.Content[valueIdx].Tag != "!!str" {
				t.Errorf("preset[%d] value %d tag = %q, want !!str",
					i, valueIdx, entry.Content[valueIdx].Tag)
			}
		}
	}
}

// TestDefaultCPPresets_FreshNodePerCall guards against callers mutating shared
// state — the patcher inserts the node into a document tree.
func TestDefaultCPPresets_FreshNodePerCall(t *testing.T) {
	first := config.DefaultCPPresets()
	first.Content[0].Content[1].Value = "mutated"

	if second := config.DefaultCPPresets(); second.Content[0].Content[1].Value != "m1.medium" {
		t.Errorf("DefaultCPPresets shares node state: got %q", second.Content[0].Content[1].Value)
	}
}
