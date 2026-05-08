package flowchart

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/luo-studio/go-mermaid/displaylist"
	"github.com/luo-studio/go-mermaid/layoutopts"
)

func TestPropertyNoPanic(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	for i := 0; i < 200; i++ {
		src := genFlowchart(rng)
		d, err := Parse(src)
		if err != nil {
			t.Fatalf("Parse(%q): %v", src, err)
		}
		dl := Layout(d, layoutopts.Options{Measurer: fixedMeasurer{}})
		if dl == nil {
			t.Fatalf("Layout returned nil for %q", src)
		}
		for _, it := range dl.Items {
			if s, ok := it.(displaylist.Shape); ok {
				if s.BBox.X < -0.001 || s.BBox.Y < -0.001 ||
					s.BBox.X+s.BBox.W > dl.Width+0.5 ||
					s.BBox.Y+s.BBox.H > dl.Height+0.5 {
					t.Fatalf("shape outside bbox: %+v in dl %vx%v (src=%q)", s, dl.Width, dl.Height, src)
				}
			}
		}
	}
}

func genFlowchart(rng *rand.Rand) string {
	dirs := []string{"TB", "BT", "LR", "RL"}
	src := fmt.Sprintf("flowchart %s\n", dirs[rng.Intn(len(dirs))])
	n := 2 + rng.Intn(5)
	ids := make([]string, n)
	for i := range ids {
		ids[i] = fmt.Sprintf("N%d", i)
	}
	for i := 1; i < n; i++ {
		op := []string{"-->", "---", "-.->", "==>"}[rng.Intn(4)]
		src += fmt.Sprintf("%s %s %s\n", ids[i-1], op, ids[i])
	}
	return src
}
