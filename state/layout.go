package state

import (
	"github.com/luo-studio/go-mermaid/autog"
	"github.com/luo-studio/go-mermaid/displaylist"
	"github.com/luo-studio/go-mermaid/internal/textutil"
	"github.com/luo-studio/go-mermaid/layoutopts"
)

// Layout positions a state diagram and returns a DisplayList. Composite
// states become clusters via the autog cluster engine.
func Layout(d *Diagram, opts layoutopts.Options) *displaylist.DisplayList {
	if d == nil || (len(d.States) == 0 && len(d.Composites) == 0) {
		return &displaylist.DisplayList{}
	}
	measurer := opts.ResolveMeasurer()
	measure := measurer.Measure

	// Composites are conceptually states, but in the autog graph they
	// are clusters. Build a set of composite IDs to exclude from the
	// node list.
	compositeID := map[string]bool{}
	var visit func(c *Composite)
	visit = func(c *Composite) {
		compositeID[c.ID] = true
		for _, ch := range c.Children {
			visit(ch)
		}
	}
	for _, c := range d.Composites {
		visit(c)
	}

	stateByID := map[string]State{}
	for _, s := range d.States {
		stateByID[s.ID] = s
	}

	var nodes []autog.Node
	for _, s := range d.States {
		if compositeID[s.ID] {
			continue
		}
		if s.IsStart || s.IsEnd {
			nodes = append(nodes, autog.Node{ID: s.ID, Width: 14, Height: 14})
			continue
		}
		w, h := measure(s.Label, displaylist.RoleStateBox)
		nodes = append(nodes, autog.Node{ID: s.ID, Width: w + 24, Height: h + 12})
	}
	var edges []autog.Edge
	for _, t := range d.Transitions {
		edges = append(edges, autog.Edge{FromID: t.From, ToID: t.To})
	}

	clusters := convertComposites(d.Composites)

	dir := autog.DirectionTB

	var (
		dl       *displaylist.DisplayList
		clusterRects []autog.ClusterRect
		laidNodes []autog.Node
		laidEdges []autog.Edge
	)

	if len(clusters) == 0 {
		out, err := autog.Layout(autog.Input{
			Nodes:        nodes,
			Edges:        edges,
			Direction:    dir,
			NodeSpacing:  opts.NodeSpacing,
			LayerSpacing: opts.LayerSpacing,
		})
		if err != nil {
			return &displaylist.DisplayList{}
		}
		dl = &displaylist.DisplayList{Width: out.Width, Height: out.Height}
		laidNodes = out.Nodes
		laidEdges = out.Edges
	} else {
		out, err := autog.LayoutClusters(autog.ClusterInput{
			Nodes:        nodes,
			Edges:        edges,
			Direction:    dir,
			NodeSpacing:  opts.NodeSpacing,
			LayerSpacing: opts.LayerSpacing,
			Clusters:     clusters,
		})
		if err != nil {
			return &displaylist.DisplayList{}
		}
		dl = &displaylist.DisplayList{Width: out.Width, Height: out.Height}
		clusterRects = out.ClusterRects
		laidNodes = out.Nodes
		laidEdges = out.Edges
	}

	emitClusterRects(dl, clusterRects)

	for _, n := range laidNodes {
		s := stateByID[n.ID]
		bbox := displaylist.Rect{X: n.X, Y: n.Y, W: n.Width, H: n.Height}
		switch {
		case s.IsStart:
			dl.Items = append(dl.Items, displaylist.Shape{
				Kind: displaylist.ShapeKindStateBullet,
				BBox: bbox,
				Role: displaylist.RolePseudostateStart,
			})
		case s.IsEnd:
			dl.Items = append(dl.Items, displaylist.Shape{
				Kind: displaylist.ShapeKindStateBullseye,
				BBox: bbox,
				Role: displaylist.RolePseudostateEnd,
			})
		default:
			dl.Items = append(dl.Items, displaylist.Shape{
				Kind: displaylist.ShapeKindRound,
				BBox: bbox,
				Role: displaylist.RoleStateBox,
			})
			dl.Items = append(dl.Items, displaylist.Text{
				Pos:    displaylist.Point{X: bbox.X + bbox.W/2, Y: bbox.Y + bbox.H/2},
				Lines:  textutil.SplitLabelLines(s.Label),
				Align:  displaylist.AlignCenter,
				VAlign: displaylist.VAlignMiddle,
				Role:   displaylist.RoleStateBox,
			})
		}
	}

	for _, e := range laidEdges {
		t := findTransition(d, e.FromID, e.ToID)
		pts := make([]displaylist.Point, len(e.Points))
		for i, p := range e.Points {
			pts[i] = displaylist.Point{X: p[0], Y: p[1]}
		}
		dl.Items = append(dl.Items, displaylist.Edge{
			Points:    pts,
			LineStyle: displaylist.LineStyleSolid,
			ArrowEnd:  displaylist.MarkerArrow,
			Role:      displaylist.RoleEdge,
		})
		if t != nil && t.Label != "" && len(pts) > 0 {
			var mid displaylist.Point
			if len(pts)%2 == 0 {
				a := pts[len(pts)/2-1]
				b := pts[len(pts)/2]
				mid = displaylist.Point{X: (a.X + b.X) / 2, Y: (a.Y + b.Y) / 2}
			} else {
				mid = pts[len(pts)/2]
			}
			dl.Items = append(dl.Items, displaylist.Text{
				Pos:    mid,
				Lines:  textutil.SplitLabelLines(t.Label),
				Align:  displaylist.AlignCenter,
				VAlign: displaylist.VAlignBottom,
				Role:   displaylist.RoleEdgeLabel,
			})
		}
	}
	return dl
}

func convertComposites(cs []*Composite) []*autog.Cluster {
	if len(cs) == 0 {
		return nil
	}
	out := make([]*autog.Cluster, len(cs))
	for i, c := range cs {
		out[i] = &autog.Cluster{
			ID:       c.ID,
			Title:    c.Label,
			NodeIDs:  c.StateIDs,
			Children: convertComposites(c.Children),
		}
	}
	return out
}

func emitClusterRects(dl *displaylist.DisplayList, rects []autog.ClusterRect) {
	for _, r := range rects {
		dl.Items = append(dl.Items, displaylist.Cluster{
			BBox:  r.BBox,
			Title: r.Title,
			Role:  displaylist.RoleStateComposite,
		})
		emitClusterRects(dl, r.Children)
	}
}

func findTransition(d *Diagram, from, to string) *Transition {
	for i := range d.Transitions {
		t := &d.Transitions[i]
		if t.From == from && t.To == to {
			return t
		}
	}
	return nil
}
