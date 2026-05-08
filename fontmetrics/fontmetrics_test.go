package fontmetrics

import (
	"math"
	"testing"

	"github.com/luo-studio/go-mermaid/displaylist"
)

func TestDefaultMeasurerNonZero(t *testing.T) {
	m := NewDefault(14)
	w, h := m.Measure("Hello", displaylist.RoleNode)
	if w <= 0 {
		t.Fatalf("width should be positive, got %v", w)
	}
	if h <= 0 {
		t.Fatalf("height should be positive, got %v", h)
	}
}

func TestDefaultMeasurerScalesWithLength(t *testing.T) {
	m := NewDefault(14)
	short, _ := m.Measure("A", displaylist.RoleNode)
	longText, _ := m.Measure("AAAAAAAAAA", displaylist.RoleNode)
	if longText < short*8 || longText > short*12 {
		t.Fatalf("scaling: got short=%v long=%v ratio=%v want 8..12", short, longText, longText/short)
	}
}

func TestDefaultMeasurerLineHeightStable(t *testing.T) {
	m := NewDefault(14)
	_, h1 := m.Measure("A", displaylist.RoleNode)
	_, h2 := m.Measure("Beethoven", displaylist.RoleNode)
	if math.Abs(h1-h2) > 0.001 {
		t.Fatalf("line height should not depend on text content: %v vs %v", h1, h2)
	}
}

func TestDefaultMeasurerBoldRole(t *testing.T) {
	m := NewDefault(14)
	regular, _ := m.Measure("Hello", displaylist.RoleNode)
	bold, _ := m.Measure("Hello", displaylist.RoleClusterTitle)
	if bold <= regular {
		t.Fatalf("bold should be wider than regular for the same text: bold=%v regular=%v", bold, regular)
	}
}
