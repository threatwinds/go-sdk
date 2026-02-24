package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNaturalLess(t *testing.T) {
	tests := []struct {
		a, b string
		want bool
	}{
		{"1.yaml", "2.yaml", true},
		{"2.yaml", "10.yaml", true},
		{"10.yaml", "2.yaml", false},
		{"12.yaml", "101.yaml", true},
		{"101.yaml", "12.yaml", false},
		{"9.yaml", "10.yaml", true},
		{"10.yaml", "9.yaml", false},
		{"1.yaml", "1.yaml", false},
		{"a.yaml", "b.yaml", true},
		{"b.yaml", "a.yaml", false},
		{"filter1.yaml", "filter2.yaml", true},
		{"filter2.yaml", "filter10.yaml", true},
		{"filter10.yaml", "filter2.yaml", false},
		{"abc.yaml", "abc.yaml", false},
		{"1a.yaml", "1b.yaml", true},
		{"1000001.yaml", "701.yaml", false},
		{"701.yaml", "1000001.yaml", true},
	}

	for _, tt := range tests {
		got := naturalLess(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("naturalLess(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestListFiles_NumericOrder(t *testing.T) {
	dir := t.TempDir()

	// Create files in intentionally non-numeric order
	names := []string{"101.yaml", "2.yaml", "12.yaml", "1.yaml", "10.yaml", "20.yaml", "3.yaml"}
	for _, name := range names {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(""), 0644); err != nil {
			t.Fatal(err)
		}
	}

	got := ListFiles(dir, ".yaml")

	expected := []string{"1.yaml", "2.yaml", "3.yaml", "10.yaml", "12.yaml", "20.yaml", "101.yaml"}
	if len(got) != len(expected) {
		t.Fatalf("got %d files, want %d", len(got), len(expected))
	}

	for i, path := range got {
		base := filepath.Base(path)
		if base != expected[i] {
			t.Errorf("index %d: got %q, want %q", i, base, expected[i])
		}
	}
}
