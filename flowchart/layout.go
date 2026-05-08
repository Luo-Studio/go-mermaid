package flowchart

import (
	"math"

	"github.com/luo-studio/go-mermaid/autog"
	"github.com/luo-studio/go-mermaid/displaylist"
	"github.com/luo-studio/go-mermaid/internal/textutil"
	"github.com/luo-studio/go-mermaid/layoutopts"
)

// Layout positions the diagram and emits a DisplayList. The Measurer
// in opts (resolved via opts.ResolveMeasurer) sizes each node's label.
//
// Diagrams without subgraphs use a flat autog layout; diagrams with
// subgraphs use the recursive cluster engine.
func Layout(d *Diagram, opts layoutopts.Options) *displaylist.DisplayList {
	if d == nil {
		return &displaylist.DisplayList{}
	}
	measurer := opts.ResolveMeasurer()

	autogNodes := make([]autog.Node, 0, len(d.Nodes))
	for _, n := range d.Nodes {
		// For multi-line labels (from <br/>), use the widest line and
		// stack heights so the box fits all rows.
		lines := splitLabelLines(n.Label)
		var lw, lh float64
		for _, ln := range lines {
			w0, h0 := measurer.Measure(ln, displaylist.RoleNode)
			if w0 > lw {
				lw = w0
			}
			lh += h0
		}
		w, h := nodeSize(n.Shape, lw, lh)
		autogNodes = append(autogNodes, autog.Node{ID: n.ID, Width: w, Height: h})
	}
	autogEdges := make([]autog.Edge, 0, len(d.Edges))
	for _, e := range d.Edges {
		autogEdges = append(autogEdges, autog.Edge{FromID: e.From, ToID: e.To})
	}

	if len(d.Subgraphs) == 0 {
		return layoutFlat(d, autogNodes, autogEdges, opts)
	}
	return layoutWithClusters(d, autogNodes, autogEdges, opts)
}

func layoutFlat(d *Diagram, autogNodes []autog.Node, autogEdges []autog.Edge, opts layoutopts.Options) *displaylist.DisplayList {
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
	emitNodes(dl, out.Nodes, d)
	emitEdges(dl, out.Edges, d)
	return dl
}

func layoutWithClusters(d *Diagram, autogNodes []autog.Node, autogEdges []autog.Edge, opts layoutopts.Options) *displaylist.DisplayList {
	clusters := convertSubgraphs(d.Subgraphs)
	out, err := autog.LayoutClusters(autog.ClusterInput{
		Direction:    autogDir(d.Direction),
		NodeSpacing:  opts.NodeSpacing,
		LayerSpacing: opts.LayerSpacing,
		Padding:      opts.Padding,
		Nodes:        autogNodes,
		Edges:        autogEdges,
		Clusters:     clusters,
	})
	if err != nil {
		return &displaylist.DisplayList{}
	}
	dl := &displaylist.DisplayList{Width: out.Width, Height: out.Height}
	emitClusterRects(dl, out.ClusterRects)
	emitNodes(dl, out.Nodes, d)
	emitEdges(dl, out.Edges, d)
	return dl
}

func convertSubgraphs(sgs []*Subgraph) []*autog.Cluster {
	out := make([]*autog.Cluster, len(sgs))
	for i, sg := range sgs {
		title := sg.Label
		if title == "" {
			title = sg.ID
		}
		out[i] = &autog.Cluster{
			ID:       sg.ID,
			Title:    title,
			NodeIDs:  sg.NodeIDs,
			Children: convertSubgraphs(sg.Children),
		}
	}
	return out
}

func emitClusterRects(dl *displaylist.DisplayList, rects []autog.ClusterRect) {
	for _, r := range rects {
		dl.Items = append(dl.Items, displaylist.Cluster{
			BBox:  r.BBox,
			Title: r.Title,
			Role:  displaylist.RoleSubgraph,
		})
		emitClusterRects(dl, r.Children)
	}
}

func emitNodes(dl *displaylist.DisplayList, nodes []autog.Node, d *Diagram) {
	astByID := map[string]Node{}
	for _, n := range d.Nodes {
		astByID[n.ID] = n
	}
	for _, n := range nodes {
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
			Lines:  splitLabelLines(ast.Label),
			Align:  displaylist.AlignCenter,
			VAlign: displaylist.VAlignMiddle,
			Role:   displaylist.RoleNode,
		})
	}
}

// splitLabelLines is a thin alias for textutil.SplitLabelLines.
var splitLabelLines = textutil.SplitLabelLines

func emitEdges(dl *displaylist.DisplayList, edges []autog.Edge, d *Diagram) {
	// Detect "anti-parallel" pairs (an edge from A→B with another B→A
	// in the same diagram) so we can offset their labels on opposite
	// sides of the line instead of stacking them at the same midpoint.
	pairKey := func(a, b string) string {
		if a < b {
			return a + "→" + b
		}
		return b + "→" + a
	}
	hasReverse := map[string]bool{}
	seen := map[string]bool{}
	for _, e := range edges {
		k := pairKey(e.FromID, e.ToID)
		if seen[k] {
			hasReverse[k] = true
		}
		seen[k] = true
	}

	for _, e := range edges {
		ast := findEdge(d, e.FromID, e.ToID)
		points := make([]displaylist.Point, 0, len(e.Points))
		for _, p := range e.Points {
			points = append(points, displaylist.Point{X: p[0], Y: p[1]})
		}
		dl.Items = append(dl.Items, displaylist.Edge{
			Points:     points,
			LineStyle:  edgeLineStyle(ast.Style),
			ArrowStart: arrowFor(ast.ArrowStart),
			ArrowEnd:   arrowFor(ast.ArrowEnd),
			Role:       displaylist.RoleEdge,
		})
		if ast.Label != "" && len(points) > 0 {
			// For anti-parallel pairs, force the side based on edge
			// direction (lex order of endpoint IDs) so the two edges
			// always pick opposite sides — independent of polyline
			// shape, which can be near-coincident for the two halves
			// of a pair.
			forceSide := 0
			if hasReverse[pairKey(e.FromID, e.ToID)] {
				if e.FromID < e.ToID {
					forceSide = 1
				} else {
					forceSide = -1
				}
			}
			pos, valign := edgeLabelPos(points, forceSide)
			dl.Items = append(dl.Items, displaylist.Text{
				Pos:    pos,
				Lines:  splitLabelLines(ast.Label),
				Align:  displaylist.AlignCenter,
				VAlign: valign,
				Role:   displaylist.RoleEdgeLabel,
			})
		}
	}
}

// edgeLabelPos returns the position to anchor an edge label and the
// vertical anchor.
//
// For single (non-anti-parallel) edges, forceSide==0: the label sits
// right at the midpoint, vertically centred on the line — looks
// natural for the typical flowchart edge.
//
// For an anti-parallel pair (bidirectional edges between the same
// two nodes), the caller passes forceSide=+1 / -1 so each edge of
// the pair lands on a different perpendicular side of the shared
// midpoint and the labels don't overlap.
func edgeLabelPos(pts []displaylist.Point, forceSide int) (displaylist.Point, displaylist.VAlign) {
	mid := midpoint(pts)
	if forceSide == 0 || len(pts) < 2 {
		return mid, displaylist.VAlignMiddle
	}
	const labelOffset = 4.5 // mm perpendicular to the line — large enough to clear ~4 mm tall text on the opposite side
	i := len(pts) / 2
	a, b := pts[i-1], pts[i]
	dx, dy := b.X-a.X, b.Y-a.Y
	length := math.Hypot(dx, dy)
	if length < 0.001 {
		return mid, displaylist.VAlignMiddle
	}
	// Perpendicular unit (rotated 90° CCW in y-down screen space).
	px, py := -dy/length, dx/length
	side := 1.0
	if forceSide < 0 {
		side = -1.0
	}
	pos := displaylist.Point{
		X: mid.X + px*labelOffset*side,
		Y: mid.Y + py*labelOffset*side,
	}
	if py*side < 0 {
		return pos, displaylist.VAlignBottom
	}
	return pos, displaylist.VAlignTop
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
	return pointAt(pts, 0.5)
}

// pointAt returns the position at fraction t along the polyline,
// where t=0 is the first point and t=1 is the last. Used to place
// edge labels off-centre when sharing an endpoint with sibling
// edges would otherwise stack labels on top of each other.
func pointAt(pts []displaylist.Point, t float64) displaylist.Point {
	if len(pts) == 0 {
		return displaylist.Point{}
	}
	if len(pts) == 1 {
		return pts[0]
	}
	// Walk the polyline measuring length to find the segment that
	// contains length*t.
	totalLen := 0.0
	segLens := make([]float64, len(pts)-1)
	for i := range segLens {
		dx := pts[i+1].X - pts[i].X
		dy := pts[i+1].Y - pts[i].Y
		segLens[i] = math.Hypot(dx, dy)
		totalLen += segLens[i]
	}
	if totalLen <= 0 {
		return pts[0]
	}
	target := totalLen * t
	walked := 0.0
	for i, sl := range segLens {
		if walked+sl >= target || i == len(segLens)-1 {
			f := 0.0
			if sl > 0 {
				f = (target - walked) / sl
			}
			return displaylist.Point{
				X: pts[i].X + (pts[i+1].X-pts[i].X)*f,
				Y: pts[i].Y + (pts[i+1].Y-pts[i].Y)*f,
			}
		}
		walked += sl
	}
	return pts[len(pts)-1]
}
