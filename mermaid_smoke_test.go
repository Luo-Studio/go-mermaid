package mermaid

import (
	"testing"

	"github.com/luo-studio/go-mermaid/displaylist"
)

func TestPhase2Smoke(t *testing.T) {
	src := "flowchart TB\nA[Start] --> B{Decide}\nB -- yes --> C[Do it]\nB -- no --> D[Skip]\n"
	dl, err := ParseAndLayout(src, LayoutOptions{})
	if err != nil {
		t.Fatalf("ParseAndLayout: %v", err)
	}
	shapes := 0
	for _, it := range dl.Items {
		if _, ok := it.(displaylist.Shape); ok {
			shapes++
		}
	}
	if shapes < 4 {
		t.Fatalf("expected >=4 shapes, got %d", shapes)
	}
}
