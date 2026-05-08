package flowchart

import (
	"testing"

	"github.com/luo-studio/go-mermaid/displaylist"
	"github.com/luo-studio/go-mermaid/layoutopts"
)

func TestLayoutSingleSubgraph(t *testing.T) {
	src := `flowchart TB
subgraph one [Outer]
A --> B
end
A --> C
`
	d, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	dl := Layout(d, layoutopts.Options{Measurer: fixedMeasurer{}})
	clusters := 0
	for _, it := range dl.Items {
		if _, ok := it.(displaylist.Cluster); ok {
			clusters++
		}
	}
	if clusters != 1 {
		t.Fatalf("expected 1 cluster, got %d", clusters)
	}
}

func TestLayoutNestedSubgraphs(t *testing.T) {
	src := `flowchart TB
subgraph outer [Outer]
A --> B
subgraph inner [Inner]
C --> D
end
B --> C
end
A --> Z
`
	d, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	dl := Layout(d, layoutopts.Options{Measurer: fixedMeasurer{}})
	clusters := 0
	for _, it := range dl.Items {
		if c, ok := it.(displaylist.Cluster); ok {
			clusters++
			if c.BBox.W <= 0 || c.BBox.H <= 0 {
				t.Errorf("cluster %q has empty bbox: %+v", c.Title, c.BBox)
			}
		}
	}
	if clusters != 2 {
		t.Fatalf("expected 2 clusters, got %d", clusters)
	}
}
