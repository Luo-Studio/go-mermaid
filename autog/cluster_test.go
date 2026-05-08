package autog

import (
	"testing"

	"github.com/luo-studio/go-mermaid/displaylist"
)

func TestClusterLayoutSingleLeafCluster(t *testing.T) {
	in := ClusterInput{
		Nodes: []Node{
			{ID: "A", Width: 30, Height: 20},
			{ID: "B", Width: 30, Height: 20},
		},
		Edges:    []Edge{{FromID: "A", ToID: "B"}},
		Clusters: []*Cluster{{ID: "G1", Title: "Group 1", NodeIDs: []string{"A", "B"}}},
	}
	out, err := LayoutClusters(in)
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}
	if len(out.ClusterRects) != 1 {
		t.Fatalf("expected 1 cluster rect, got %d", len(out.ClusterRects))
	}
	cr := out.ClusterRects[0]
	if cr.BBox.W <= 0 || cr.BBox.H <= 0 {
		t.Fatalf("cluster bbox empty: %+v", cr)
	}
	for _, n := range out.Nodes {
		mid := displaylist.Point{X: n.X + n.Width/2, Y: n.Y + n.Height/2}
		if !cr.BBox.Contains(mid) {
			t.Errorf("node %s outside cluster bbox: node=%+v cluster=%+v", n.ID, n, cr)
		}
	}
}

func TestClusterLayoutNested(t *testing.T) {
	in := ClusterInput{
		Nodes: []Node{
			{ID: "A", Width: 30, Height: 20},
			{ID: "B", Width: 30, Height: 20},
			{ID: "C", Width: 30, Height: 20},
		},
		Edges: []Edge{
			{FromID: "A", ToID: "B"},
			{FromID: "B", ToID: "C"},
		},
		Clusters: []*Cluster{{
			ID:      "outer",
			Title:   "Outer",
			NodeIDs: []string{"A"},
			Children: []*Cluster{{
				ID:      "inner",
				Title:   "Inner",
				NodeIDs: []string{"B", "C"},
			}},
		}},
	}
	out, err := LayoutClusters(in)
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}
	if len(out.ClusterRects) != 1 {
		t.Fatalf("expected 1 root cluster rect, got %d", len(out.ClusterRects))
	}
	outer := out.ClusterRects[0]
	if outer.ID != "outer" {
		t.Errorf("outer ID: %q", outer.ID)
	}
	if len(outer.Children) != 1 {
		t.Fatalf("outer should have 1 child, got %d", len(outer.Children))
	}
	inner := outer.Children[0]
	if inner.ID != "inner" {
		t.Errorf("inner ID: %q", inner.ID)
	}
	// Inner bbox must lie within outer bbox.
	if inner.BBox.X < outer.BBox.X-0.001 || inner.BBox.Y < outer.BBox.Y-0.001 ||
		inner.BBox.X+inner.BBox.W > outer.BBox.X+outer.BBox.W+0.001 ||
		inner.BBox.Y+inner.BBox.H > outer.BBox.Y+outer.BBox.H+0.001 {
		t.Errorf("inner not contained in outer: inner=%+v outer=%+v", inner.BBox, outer.BBox)
	}
}

func TestClusterLayoutMixedClusterAndUnclustered(t *testing.T) {
	in := ClusterInput{
		Nodes: []Node{
			{ID: "X", Width: 30, Height: 20},
			{ID: "A", Width: 30, Height: 20},
			{ID: "B", Width: 30, Height: 20},
		},
		Edges: []Edge{
			{FromID: "X", ToID: "A"},
			{FromID: "A", ToID: "B"},
		},
		Clusters: []*Cluster{{ID: "G", Title: "G", NodeIDs: []string{"A", "B"}}},
	}
	out, err := LayoutClusters(in)
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}
	if len(out.Nodes) != 3 {
		t.Fatalf("expected 3 positioned nodes, got %d", len(out.Nodes))
	}
	if len(out.ClusterRects) != 1 {
		t.Fatalf("expected 1 cluster rect, got %d", len(out.ClusterRects))
	}
	// X should NOT be inside the cluster bbox.
	cr := out.ClusterRects[0]
	for _, n := range out.Nodes {
		if n.ID != "X" {
			continue
		}
		mid := displaylist.Point{X: n.X + n.Width/2, Y: n.Y + n.Height/2}
		if cr.BBox.Contains(mid) {
			t.Errorf("unclustered node X should be outside cluster bbox; node=%+v cluster=%+v", n, cr)
		}
	}
}
