package mermaid

import (
	"errors"
	"testing"

	"github.com/luo-studio/go-mermaid/displaylist"
)

func TestDetectType(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want diagramType
	}{
		{"flowchart-TB", "flowchart TB\n  A --> B\n", typeFlowchart},
		{"graph-LR", "graph LR\n  A --> B\n", typeFlowchart},
		{"sequence", "sequenceDiagram\n  A->>B: hi\n", typeSequence},
		{"class", "classDiagram\n  A <|-- B\n", typeClass},
		{"er", "erDiagram\n  A ||--o{ B : has\n", typeER},
		{"state-v2", "stateDiagram-v2\n  [*] --> S\n", typeState},
		{"state", "stateDiagram\n  [*] --> S\n", typeState},
		{"unknown", "garbage\n", typeUnknown},
		{"empty", "", typeUnknown},
		{"comments-then-flowchart", "%% comment\nflowchart TB\nA --> B", typeFlowchart},
		{"leading-blank-lines", "\n\nflowchart TB\nA --> B", typeFlowchart},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := detectType(tc.src); got != tc.want {
				t.Errorf("detectType(%q) = %v, want %v", tc.src, got, tc.want)
			}
		})
	}
}

func TestParseAndLayoutUnknownDiagram(t *testing.T) {
	_, err := ParseAndLayout("not a diagram", LayoutOptions{})
	if !errors.Is(err, ErrUnknownDiagram) {
		t.Fatalf("expected ErrUnknownDiagram, got %v", err)
	}
}

// Once phases land, individual diagram types stop returning
// ErrNotImplemented. We assert per-type below.

func TestParseAndLayoutFlowchart(t *testing.T) {
	dl, err := ParseAndLayout("flowchart TB\nA --> B\n", LayoutOptions{})
	if err != nil {
		t.Fatalf("ParseAndLayout: %v", err)
	}
	if dl == nil || len(dl.Items) == 0 {
		t.Fatal("expected non-empty DisplayList")
	}
}

func TestParseAndLayoutSequence(t *testing.T) {
	dl, err := ParseAndLayout("sequenceDiagram\nparticipant A\nparticipant B\nA->>B: hi\n", LayoutOptions{})
	if err != nil {
		t.Fatalf("ParseAndLayout: %v", err)
	}
	if dl == nil || len(dl.Items) == 0 {
		t.Fatal("expected non-empty DisplayList for sequence")
	}
}

func TestParseAndLayoutClass(t *testing.T) {
	dl, err := ParseAndLayout("classDiagram\nclass A\nclass B\nA <|-- B\n", LayoutOptions{})
	if err != nil {
		t.Fatalf("ParseAndLayout: %v", err)
	}
	if dl == nil || len(dl.Items) == 0 {
		t.Fatal("expected non-empty DisplayList for class")
	}
}

func TestParseAndLayoutER(t *testing.T) {
	dl, err := ParseAndLayout("erDiagram\nA ||--o{ B : has\n", LayoutOptions{})
	if err != nil {
		t.Fatalf("ParseAndLayout: %v", err)
	}
	if dl == nil || len(dl.Items) == 0 {
		t.Fatal("expected non-empty DisplayList for ER")
	}
}

func TestParseAndLayoutState(t *testing.T) {
	dl, err := ParseAndLayout("stateDiagram-v2\n[*] --> S\nS --> [*]\n", LayoutOptions{})
	if err != nil {
		t.Fatalf("ParseAndLayout: %v", err)
	}
	if dl == nil || len(dl.Items) == 0 {
		t.Fatal("expected non-empty DisplayList for state")
	}
}

func TestLayoutOptionsZeroValueValid(t *testing.T) {
	var opts LayoutOptions
	if opts.Measurer != nil {
		t.Fatal("zero LayoutOptions should have nil Measurer")
	}
}

func TestLayoutOptionsDefaultMeasurerNotNil(t *testing.T) {
	var opts LayoutOptions
	m := opts.ResolveMeasurer()
	if m == nil {
		t.Fatal("opts.ResolveMeasurer() must never return nil")
	}
	w, h := m.Measure("Hello", "")
	if w <= 0 || h <= 0 {
		t.Fatalf("default measurer should report positive metrics, got %vx%v", w, h)
	}
}

func TestLayoutOptionsCustomMeasurerWins(t *testing.T) {
	called := false
	custom := measurerFunc(func(text string, role displaylist.Role) (float64, float64) {
		called = true
		return 42, 42
	})
	opts := LayoutOptions{Measurer: custom}
	w, h := opts.ResolveMeasurer().Measure("anything", "")
	if !called || w != 42 || h != 42 {
		t.Fatalf("custom measurer was not preferred: called=%v w=%v h=%v", called, w, h)
	}
}

type measurerFunc func(text string, role displaylist.Role) (float64, float64)

func (f measurerFunc) Measure(text string, role displaylist.Role) (float64, float64) {
	return f(text, role)
}
