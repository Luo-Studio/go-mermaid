package flowchart

import (
	"testing"

	"github.com/luo-studio/go-mermaid/displaylist"
	"github.com/luo-studio/go-mermaid/layoutopts"
)

func TestLayoutSimpleChain(t *testing.T) {
	d, _ := Parse("flowchart TB\nA --> B\nB --> C\n")
	dl := Layout(d, layoutopts.Options{Measurer: fixedMeasurer{}})
	if dl == nil {
		t.Fatal("Layout returned nil")
	}
	shapeCount, edgeCount := 0, 0
	for _, it := range dl.Items {
		switch it.(type) {
		case displaylist.Shape:
			shapeCount++
		case displaylist.Edge:
			edgeCount++
		}
	}
	if shapeCount != 3 {
		t.Errorf("shapes: got %d want 3", shapeCount)
	}
	if edgeCount != 2 {
		t.Errorf("edges: got %d want 2", edgeCount)
	}
	if dl.Width <= 0 || dl.Height <= 0 {
		t.Errorf("dl bbox: %vx%v", dl.Width, dl.Height)
	}
}

func TestLayoutEdgeLabel(t *testing.T) {
	d, _ := Parse("flowchart TB\nA -- yes --> B\n")
	dl := Layout(d, layoutopts.Options{Measurer: fixedMeasurer{}})
	hasLabel := false
	for _, it := range dl.Items {
		if t, ok := it.(displaylist.Text); ok && t.Role == displaylist.RoleEdgeLabel {
			if len(t.Lines) > 0 && t.Lines[0] == "yes" {
				hasLabel = true
			}
		}
	}
	if !hasLabel {
		t.Errorf("expected edge label 'yes' in DisplayList; got items: %+v", dl.Items)
	}
}

func TestLayoutAllShapesEmits(t *testing.T) {
	src := `flowchart TB
A[Rect]
B(Round)
C([Stadium])
D{Diamond}
E((Circle))
F[(Cylinder)]
G{{Hex}}
A --> B
B --> C
C --> D
D --> E
E --> F
F --> G
`
	d, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	dl := Layout(d, layoutopts.Options{Measurer: fixedMeasurer{}})
	shapes := 0
	for _, it := range dl.Items {
		if _, ok := it.(displaylist.Shape); ok {
			shapes++
		}
	}
	if shapes != 7 {
		t.Errorf("expected 7 shapes, got %d", shapes)
	}
}

type fixedMeasurer struct{}

func (fixedMeasurer) Measure(text string, role displaylist.Role) (float64, float64) {
	return float64(len(text)) * 7, 14
}
