package autog

import "testing"

func TestLayoutChainOfThree(t *testing.T) {
	in := Input{
		Nodes: []Node{
			{ID: "A", Width: 30, Height: 20},
			{ID: "B", Width: 30, Height: 20},
			{ID: "C", Width: 30, Height: 20},
		},
		Edges: []Edge{
			{FromID: "A", ToID: "B"},
			{FromID: "B", ToID: "C"},
		},
		Direction:    DirectionTB,
		NodeSpacing:  10,
		LayerSpacing: 30,
	}

	out, err := Layout(in)
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}
	if len(out.Nodes) != 3 {
		t.Fatalf("expected 3 positioned nodes, got %d", len(out.Nodes))
	}
	if len(out.Edges) != 2 {
		t.Fatalf("expected 2 positioned edges, got %d", len(out.Edges))
	}

	posOf := func(id string) Node {
		for _, n := range out.Nodes {
			if n.ID == id {
				return n
			}
		}
		t.Fatalf("node %q not in layout", id)
		return Node{}
	}
	a, b, c := posOf("A"), posOf("B"), posOf("C")
	if !(a.Y < b.Y && b.Y < c.Y) {
		t.Fatalf("expected A.Y < B.Y < C.Y, got %v %v %v", a.Y, b.Y, c.Y)
	}

	if out.Width <= 0 || out.Height <= 0 {
		t.Fatalf("expected positive bbox, got %vx%v", out.Width, out.Height)
	}
}

func TestLayoutTwoNodesOneEdge(t *testing.T) {
	in := Input{
		Nodes: []Node{
			{ID: "A", Width: 30, Height: 20},
			{ID: "B", Width: 30, Height: 20},
		},
		Edges: []Edge{{FromID: "A", ToID: "B"}},
	}
	out, err := Layout(in)
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}
	if len(out.Nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(out.Nodes))
	}
}

func TestLayoutEmpty(t *testing.T) {
	out, err := Layout(Input{})
	if err != nil {
		t.Fatalf("Layout(empty): %v", err)
	}
	if len(out.Nodes) != 0 || len(out.Edges) != 0 {
		t.Fatalf("expected empty layout, got %v", out)
	}
	if out.Width != 0 || out.Height != 0 {
		t.Fatalf("expected zero bbox, got %vx%v", out.Width, out.Height)
	}
}

func TestLayoutIsolatedNode(t *testing.T) {
	// Single node with no edges. Adapter must inject a sentinel
	// self-loop and drop it from output.
	in := Input{
		Nodes: []Node{{ID: "solo", Width: 30, Height: 20}},
	}
	out, err := Layout(in)
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}
	if len(out.Nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(out.Nodes))
	}
	if len(out.Edges) != 0 {
		t.Fatalf("expected 0 edges, got %d", len(out.Edges))
	}
}

func TestLayoutLR(t *testing.T) {
	in := Input{
		Nodes: []Node{
			{ID: "A", Width: 30, Height: 20},
			{ID: "B", Width: 30, Height: 20},
		},
		Edges:     []Edge{{FromID: "A", ToID: "B"}},
		Direction: DirectionLR,
	}
	out, err := Layout(in)
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}
	posOf := func(id string) Node {
		for _, n := range out.Nodes {
			if n.ID == id {
				return n
			}
		}
		t.Fatalf("node %q missing", id)
		return Node{}
	}
	a, b := posOf("A"), posOf("B")
	if !(a.X < b.X) {
		t.Fatalf("expected A.X < B.X in LR mode, got %v vs %v", a.X, b.X)
	}
}
