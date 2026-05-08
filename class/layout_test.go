package class

import (
	"testing"

	"github.com/luo-studio/go-mermaid/displaylist"
	"github.com/luo-studio/go-mermaid/layoutopts"
)

func TestLayoutBasic(t *testing.T) {
	src := `classDiagram
class Animal {
+name : string
+eat() void
}
class Dog
Animal <|-- Dog
`
	d, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	dl := Layout(d, layoutopts.Options{Measurer: fixedMeasurer{}})
	if dl.Width <= 0 || dl.Height <= 0 {
		t.Fatalf("bbox: %vx%v", dl.Width, dl.Height)
	}
	classBoxes := 0
	for _, it := range dl.Items {
		if s, ok := it.(displaylist.Shape); ok && s.Role == displaylist.RoleClassBox {
			classBoxes++
		}
	}
	if classBoxes != 2 {
		t.Errorf("class boxes: %d", classBoxes)
	}
}

func TestLayoutWithNamespace(t *testing.T) {
	src := `classDiagram
namespace Net {
class HTTP
class TCP
}
HTTP --> TCP
`
	d, _ := Parse(src)
	dl := Layout(d, layoutopts.Options{Measurer: fixedMeasurer{}})
	clusters := 0
	for _, it := range dl.Items {
		if _, ok := it.(displaylist.Cluster); ok {
			clusters++
		}
	}
	if clusters != 1 {
		t.Errorf("expected 1 namespace cluster, got %d", clusters)
	}
}

type fixedMeasurer struct{}

func (fixedMeasurer) Measure(text string, role displaylist.Role) (float64, float64) {
	return float64(len(text)) * 7, 14
}
