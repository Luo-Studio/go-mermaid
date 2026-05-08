package flowchart

import (
	"github.com/luo-studio/go-mermaid/autog"
	"github.com/luo-studio/go-mermaid/displaylist"
	"github.com/luo-studio/go-mermaid/layoutopts"
)

// Layout positions the diagram and emits a DisplayList. The Measurer
// in opts (resolved via opts.ResolveMeasurer) sizes each node's label.
//
// Phase 2: subgraphs are parsed but flattened in layout — every
// node is laid out at top level. Cluster recursion lands in Phase 3.
func Layout(d *Diagram, opts layoutopts.Options) *displaylist.DisplayList {
	if d == nil {
		return &displaylist.DisplayList{}
	}
	measurer := opts.ResolveMeasurer()

	autogNodes := make([]autog.Node, 0, len(d.Nodes))
	for _, n := range d.Nodes {
		lw, lh := measurer.Measure(n.Label, displaylist.RoleNode)
		w, h := nodeSize(n.Shape, lw, lh)
		autogNodes = append(autogNodes, autog.Node{ID: n.ID, Width: w, Height: h})
	}
	autogEdges := make([]autog.Edge, 0, len(d.Edges))
	for _, e := range d.Edges {
		autogEdges = append(autogEdges, autog.Edge{FromID: e.From, ToID: e.To})
	}

	out, err := autog.Layout(autog.Input{
		Nodes:        autogNodes,
		Edges:        autogEdges,
		Direction:    autogDir(d.Direction),
		NodeSpacing:  opts.NodeSpacing,
		LayerSpacing: opts.LayerSpacing,
		Padding:      opts.Padding,
	})
	if err != nil {
		return &displaylist.DisplayList{}
	}

	dl := &displaylist.DisplayList{Width: out.Width, Height: out.Height}
	posByID := map[string]autog.Node{}
	for _, n := range out.Nodes {
		posByID[n.ID] = n
	}
	astByID := map[string]Node{}
	for _, n := range d.Nodes {
		astByID[n.ID] = n
	}

	for _, n := range out.Nodes {
		ast := astByID[n.ID]
		bbox := displaylist.Rect{X: n.X, Y: n.Y, W: n.Width, H: n.Height}
		shape := displaylist.Shape{
			Kind: shapeKind(ast.Shape),
			BBox: bbox,
			Role: displaylist.RoleNode,
		}
		if shape.Kind == displaylist.ShapeKindCustom {
			shape.Path = customPath(ast.Shape, bbox)
		}
		dl.Items = append(dl.Items, shape)
		dl.Items = append(dl.Items, displaylist.Text{
			Pos:    displaylist.Point{X: bbox.X + bbox.W/2, Y: bbox.Y + bbox.H/2},
			Lines:  []string{ast.Label},
			Align:  displaylist.AlignCenter,
			VAlign: displaylist.VAlignMiddle,
			Role:   displaylist.RoleNode,
		})
	}

	for _, e := range out.Edges {
		points := make([]displaylist.Point, 0, len(e.Points))
		for _, p := range e.Points {
			points = append(points, displaylist.Point{X: p[0], Y: p[1]})
		}
		ast := findEdge(d, e.FromID, e.ToID)
		dl.Items = append(dl.Items, displaylist.Edge{
			Points:     points,
			LineStyle:  edgeLineStyle(ast.Style),
			ArrowStart: arrowFor(ast.ArrowStart),
			ArrowEnd:   arrowFor(ast.ArrowEnd),
			Role:       displaylist.RoleEdge,
		})
		if ast.Label != "" {
			dl.Items = append(dl.Items, displaylist.Text{
				Pos:    midpoint(points),
				Lines:  []string{ast.Label},
				Align:  displaylist.AlignCenter,
				VAlign: displaylist.VAlignMiddle,
				Role:   displaylist.RoleEdgeLabel,
			})
		}
	}
	return dl
}

func autogDir(d Direction) autog.Direction {
	switch d {
	case DirectionBT:
		return autog.DirectionBT
	case DirectionLR:
		return autog.DirectionLR
	case DirectionRL:
		return autog.DirectionRL
	default:
		return autog.DirectionTB
	}
}

func edgeLineStyle(s EdgeStyle) displaylist.LineStyle {
	switch s {
	case EdgeDotted:
		return displaylist.LineStyleDotted
	case EdgeThick:
		return displaylist.LineStyleThick
	default:
		return displaylist.LineStyleSolid
	}
}

func arrowFor(present bool) displaylist.MarkerKind {
	if present {
		return displaylist.MarkerArrow
	}
	return displaylist.MarkerNone
}

func findEdge(d *Diagram, from, to string) Edge {
	for _, e := range d.Edges {
		if e.From == from && e.To == to {
			return e
		}
	}
	return Edge{}
}

func midpoint(pts []displaylist.Point) displaylist.Point {
	if len(pts) == 0 {
		return displaylist.Point{}
	}
	if len(pts) == 1 {
		return pts[0]
	}
	mid := len(pts) / 2
	return displaylist.Point{X: (pts[mid-1].X + pts[mid].X) / 2, Y: (pts[mid-1].Y + pts[mid].Y) / 2}
}
