package sequence

import (
	"testing"

	"github.com/luo-studio/go-mermaid/displaylist"
	"github.com/luo-studio/go-mermaid/layoutopts"
)

func TestLayoutBasic(t *testing.T) {
	src := `sequenceDiagram
participant A as Alice
participant B as Bob
A->>B: Hi
B-->>A: Hello
`
	d, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	dl := Layout(d, layoutopts.Options{Measurer: fixedMeasurer{}})
	if dl.Width <= 0 || dl.Height <= 0 {
		t.Fatalf("bbox: %vx%v", dl.Width, dl.Height)
	}
	actorBoxes := 0
	lifelines := 0
	messages := 0
	for _, it := range dl.Items {
		switch v := it.(type) {
		case displaylist.Shape:
			if v.Role == displaylist.RoleActorBox {
				actorBoxes++
			}
		case displaylist.Edge:
			if v.Role == displaylist.RoleLifeline {
				lifelines++
			}
			if v.Role == displaylist.RoleEdge {
				messages++
			}
		}
	}
	// 2 actors × 2 (header + footer) = 4 boxes
	if actorBoxes != 4 {
		t.Errorf("actor boxes: got %d want 4", actorBoxes)
	}
	if lifelines != 2 {
		t.Errorf("lifelines: got %d want 2", lifelines)
	}
	if messages != 2 {
		t.Errorf("messages: got %d want 2", messages)
	}
}

func TestLayoutBlock(t *testing.T) {
	src := `sequenceDiagram
participant A
participant B
loop Every minute
A->>B: ping
end
`
	d, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	dl := Layout(d, layoutopts.Options{Measurer: fixedMeasurer{}})
	clusters := 0
	for _, it := range dl.Items {
		if c, ok := it.(displaylist.Cluster); ok {
			if c.Role == displaylist.RoleLoopBlock {
				clusters++
			}
		}
	}
	if clusters != 1 {
		t.Errorf("expected 1 loop cluster, got %d", clusters)
	}
}

func TestLayoutNote(t *testing.T) {
	src := `sequenceDiagram
participant A
Note left of A: hello
`
	d, _ := Parse(src)
	dl := Layout(d, layoutopts.Options{Measurer: fixedMeasurer{}})
	notes := 0
	for _, it := range dl.Items {
		if c, ok := it.(displaylist.Cluster); ok {
			if c.Role == displaylist.RoleSequenceNote {
				notes++
			}
		}
	}
	if notes != 1 {
		t.Errorf("expected 1 note cluster, got %d", notes)
	}
}

type fixedMeasurer struct{}

func (fixedMeasurer) Measure(text string, role displaylist.Role) (float64, float64) {
	return float64(len(text)) * 7, 14
}
