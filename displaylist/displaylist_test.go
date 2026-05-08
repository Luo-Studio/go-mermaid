package displaylist

import (
	"encoding/json"
	"testing"
)

func TestShapeImplementsItem(t *testing.T) {
	var _ Item = Shape{}
	var _ Item = Edge{}
	var _ Item = Text{}
	var _ Item = Cluster{}
	var _ Item = Marker{}
}

func TestRectContains(t *testing.T) {
	r := Rect{X: 10, Y: 10, W: 20, H: 20}
	if !r.Contains(Point{X: 15, Y: 15}) {
		t.Fatal("Rect should contain its interior point")
	}
	if r.Contains(Point{X: 5, Y: 15}) {
		t.Fatal("Rect should not contain a point left of it")
	}
	if r.Contains(Point{X: 30, Y: 15}) {
		t.Fatal("Rect should not contain a point at its right edge (half-open)")
	}
}

func TestDisplayListZeroValue(t *testing.T) {
	var dl DisplayList
	if len(dl.Items) != 0 {
		t.Fatal("zero-value DisplayList should have no items")
	}
	if dl.Width != 0 || dl.Height != 0 {
		t.Fatal("zero-value DisplayList should have zero dimensions")
	}
}

func TestDisplayListJSONRoundTrip(t *testing.T) {
	in := DisplayList{
		Width:  100,
		Height: 50,
		Items: []Item{
			Shape{Kind: ShapeKindRect, BBox: Rect{X: 0, Y: 0, W: 30, H: 20}, Role: RoleNode},
			Edge{
				Points:    []Point{{X: 30, Y: 10}, {X: 60, Y: 10}},
				LineStyle: LineStyleSolid,
				ArrowEnd:  MarkerArrow,
				Role:      RoleEdge,
			},
			Text{Pos: Point{X: 15, Y: 10}, Lines: []string{"A"}, Align: AlignCenter, VAlign: VAlignMiddle, Role: RoleNode},
			Cluster{BBox: Rect{X: 0, Y: 0, W: 100, H: 50}, Title: "outer", Role: RoleSubgraph},
			Marker{Pos: Point{X: 60, Y: 10}, Angle: 0, Kind: MarkerArrow, Role: RoleEdge},
		},
	}

	data, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var out DisplayList
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if out.Width != in.Width || out.Height != in.Height {
		t.Fatalf("dims: got %v×%v want %v×%v", out.Width, out.Height, in.Width, in.Height)
	}
	if len(out.Items) != len(in.Items) {
		t.Fatalf("item count: got %d want %d", len(out.Items), len(in.Items))
	}
	for i := range in.Items {
		if in.Items[i].itemKind() != out.Items[i].itemKind() {
			t.Fatalf("item %d kind: got %s want %s", i, out.Items[i].itemKind(), in.Items[i].itemKind())
		}
	}
}
