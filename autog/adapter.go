// Package autog wraps github.com/Luo-Studio/autog with a smaller surface
// suited to go-mermaid's layout stage. It is a thin adapter — no
// cluster recursion (added in Phase 3), no per-diagram-type
// knowledge.
package autog

import (
	"fmt"
	"math"
	"runtime"

	upstream "github.com/Luo-Studio/autog"
	upgraph "github.com/Luo-Studio/autog/graph"
)

// Direction matches Mermaid's flowchart direction.
type Direction int

const (
	DirectionTB Direction = iota
	DirectionBT
	DirectionLR
	DirectionRL
)

// Node is a layout input/output node. Width and Height are in
// DisplayList units; X and Y are populated by Layout.
type Node struct {
	ID            string
	Width, Height float64
	X, Y          float64
}

// Edge is a layout input/output edge. After Layout, Points is the
// polyline waypoints in DisplayList units.
type Edge struct {
	FromID string
	ToID   string
	Points [][2]float64
}

// Input is the cumulative layout request.
type Input struct {
	Nodes        []Node
	Edges        []Edge
	Direction    Direction
	NodeSpacing  float64
	LayerSpacing float64
	Padding      float64
}

// Output carries the positioned graph.
type Output struct {
	Width, Height float64
	Nodes         []Node
	Edges         []Edge
}

// Layout runs the autog pipeline. Returns an empty Output if Input
// has no nodes. Wraps autog panics into errors so a malformed input
// doesn't crash the caller; runtime panics are re-raised.
func Layout(in Input) (out Output, err error) {
	if len(in.Nodes) == 0 {
		return Output{}, nil
	}

	defer func() {
		if r := recover(); r != nil {
			if _, isRuntime := r.(runtime.Error); isRuntime {
				panic(r)
			}
			err = fmt.Errorf("autog: layout panic: %v", r)
		}
	}()

	// Defaults are tuned for ~10pt body text in mm (the typical fpdf
	// case). Larger callers can override via Input.NodeSpacing /
	// LayerSpacing. Previous defaults (24/40) were sized for the old
	// fontSize=14 layout that produced ~30-mm tall nodes; with the
	// current ~10-mm-tall nodes they looked proportionally huge.
	nodeSpacing := in.NodeSpacing
	if nodeSpacing == 0 {
		nodeSpacing = 14
	}
	layerSpacing := in.LayerSpacing
	if layerSpacing == 0 {
		layerSpacing = 20
	}

	// autog discovers nodes via edge endpoints. To support isolated
	// nodes, emit a self-loop sentinel for any node that doesn't
	// appear in the edge set. autog drops self-loops during layout
	// (preprocessor.IgnoreSelfLoops) so they don't pollute the result.
	endpointSeen := make(map[string]bool, len(in.Nodes)*2)
	for _, e := range in.Edges {
		endpointSeen[e.FromID] = true
		endpointSeen[e.ToID] = true
	}
	adj := make([][]string, 0, len(in.Edges)+len(in.Nodes))
	for _, e := range in.Edges {
		adj = append(adj, []string{e.FromID, e.ToID})
	}
	for _, n := range in.Nodes {
		if !endpointSeen[n.ID] {
			adj = append(adj, []string{n.ID, n.ID})
		}
	}
	src := upgraph.EdgeSlice(adj)

	// autog only computes top-to-bottom layouts internally. For LR/RL
	// we rotate the result, but autog's per-rank and inter-rank
	// spacing uses node Width/Height as it sees them — and after
	// rotation, what we want as the horizontal axis was autog's
	// vertical axis. Pre-swap W and H so autog spaces with the
	// dimensions that match the *post-rotation* axis usage.
	swapInputDims := in.Direction == DirectionLR || in.Direction == DirectionRL
	sizes := make(map[string]upgraph.Size, len(in.Nodes))
	for _, n := range in.Nodes {
		w, h := n.Width, n.Height
		if swapInputDims {
			w, h = h, w
		}
		sizes[n.ID] = upgraph.Size{W: w, H: h}
	}

	layout := upstream.Layout(
		src,
		upstream.WithNodeSize(sizes),
		upstream.WithNodeSpacing(nodeSpacing),
		upstream.WithLayerSpacing(layerSpacing),
		// BrandesKoepf averages four alignment passes (left-up,
		// left-down, right-up, right-down). It centres parents over
		// their children's bounding box more reliably than
		// NetworkSimplex (which can still bias toward the leftmost
		// child) and avoids SinkColoring's strong left-bias for
		// fan-outs. Any chain alignment offsets are masked by
		// orthogonal edge routing below.
		upstream.WithPositioning(upstream.PositioningBrandesKoepf),
		// Spline edge routing: piece-wise cubic Bézier curves that
		// route around obstacle nodes natively. Each edge comes back
		// as 1+3N points (start, then groups of two control points
		// + an endpoint per Bézier segment).
		upstream.WithEdgeRouting(upstream.EdgeRoutingSplines),
	)

	// Determine post-layout coordinate flip for direction. autog's
	// default is top-to-bottom; for the other directions we transform
	// once we know the bbox.
	//
	// Track min as well as max — NetworkSimplex (and some other
	// positioning algorithms) can emit negative coordinates for nodes
	// the algorithm centres relative to siblings (e.g. a parent placed
	// above its first child whose Y is then driven negative). If we
	// only tracked max we'd report a too-small bbox and the renderer
	// would draw above its claimed top-left corner — overlapping
	// surrounding content. Shift everything so the bbox starts at (0, 0).
	minX, minY := math.Inf(1), math.Inf(1)
	maxX, maxY := math.Inf(-1), math.Inf(-1)

	out.Nodes = make([]Node, 0, len(layout.Nodes))
	for _, n := range layout.Nodes {
		nn := Node{
			ID:     n.ID,
			Width:  n.Size.W,
			Height: n.Size.H,
			X:      n.Size.X,
			Y:      n.Size.Y,
		}
		out.Nodes = append(out.Nodes, nn)
		if nn.X < minX {
			minX = nn.X
		}
		if nn.Y < minY {
			minY = nn.Y
		}
		if rx := nn.X + nn.Width; rx > maxX {
			maxX = rx
		}
		if ry := nn.Y + nn.Height; ry > maxY {
			maxY = ry
		}
	}
	out.Edges = make([]Edge, 0, len(layout.Edges))
	for _, e := range layout.Edges {
		if e.FromID == e.ToID {
			// drop sentinel self-loops we injected for isolated nodes
			continue
		}
		ee := Edge{FromID: e.FromID, ToID: e.ToID, Points: append([][2]float64{}, e.Points...)}
		out.Edges = append(out.Edges, ee)
		for _, p := range ee.Points {
			if p[0] < minX {
				minX = p[0]
			}
			if p[1] < minY {
				minY = p[1]
			}
			if p[0] > maxX {
				maxX = p[0]
			}
			if p[1] > maxY {
				maxY = p[1]
			}
		}
	}

	if minX > 0 {
		minX = 0
	}
	if minY > 0 {
		minY = 0
	}
	if minX < 0 || minY < 0 {
		for i := range out.Nodes {
			out.Nodes[i].X -= minX
			out.Nodes[i].Y -= minY
		}
		for i := range out.Edges {
			for j := range out.Edges[i].Points {
				out.Edges[i].Points[j][0] -= minX
				out.Edges[i].Points[j][1] -= minY
			}
		}
		maxX -= minX
		maxY -= minY
	}

	// Apply direction transform. autog's default = TB; transform the
	// coordinates of nodes and edge points in-place.
	transformDirection(in.Direction, &out, maxX, maxY)


	out.Width = out.Width + in.Padding*2
	out.Height = out.Height + in.Padding*2
	if in.Padding != 0 {
		for i := range out.Nodes {
			out.Nodes[i].X += in.Padding
			out.Nodes[i].Y += in.Padding
		}
		for i := range out.Edges {
			for j := range out.Edges[i].Points {
				out.Edges[i].Points[j][0] += in.Padding
				out.Edges[i].Points[j][1] += in.Padding
			}
		}
	}
	return out, nil
}

// bboxOf returns the (width, height) bounding all positioned nodes
// and edge waypoints in out. Use after a coordinate transformation
// to recompute the diagram extents.
func bboxOf(out *Output) (float64, float64) {
	var w, h float64
	for _, n := range out.Nodes {
		if rx := n.X + n.Width; rx > w {
			w = rx
		}
		if ry := n.Y + n.Height; ry > h {
			h = ry
		}
	}
	for _, e := range out.Edges {
		for _, p := range e.Points {
			if p[0] > w {
				w = p[0]
			}
			if p[1] > h {
				h = p[1]
			}
		}
	}
	return w, h
}

func transformDirection(dir Direction, out *Output, maxX, maxY float64) {
	switch dir {
	case DirectionTB:
		out.Width = maxX
		out.Height = maxY
	case DirectionBT:
		// flip Y
		for i := range out.Nodes {
			out.Nodes[i].Y = maxY - out.Nodes[i].Y - out.Nodes[i].Height
		}
		for i := range out.Edges {
			for j := range out.Edges[i].Points {
				out.Edges[i].Points[j][1] = maxY - out.Edges[i].Points[j][1]
			}
		}
		out.Width = maxX
		out.Height = maxY
	case DirectionLR:
		// Input was given to autog with W/H pre-swapped (so autog's
		// rank-spacing — which it applies to its Y axis — uses what
		// the caller calls "Width", giving us the column gap we want
		// after rotation). Swap (X, Y) of positions and edge points
		// to rotate the layout 90° CCW, then swap node W/H back so
		// the output dimensions match what the caller passed in.
		for i := range out.Nodes {
			out.Nodes[i].X, out.Nodes[i].Y = out.Nodes[i].Y, out.Nodes[i].X
			out.Nodes[i].Width, out.Nodes[i].Height = out.Nodes[i].Height, out.Nodes[i].Width
		}
		for i := range out.Edges {
			for j := range out.Edges[i].Points {
				out.Edges[i].Points[j][0], out.Edges[i].Points[j][1] = out.Edges[i].Points[j][1], out.Edges[i].Points[j][0]
			}
		}
		out.Width, out.Height = bboxOf(out)
	case DirectionRL:
		// Same as LR, then mirror X.
		for i := range out.Nodes {
			out.Nodes[i].X, out.Nodes[i].Y = out.Nodes[i].Y, out.Nodes[i].X
			out.Nodes[i].Width, out.Nodes[i].Height = out.Nodes[i].Height, out.Nodes[i].Width
		}
		for i := range out.Edges {
			for j := range out.Edges[i].Points {
				out.Edges[i].Points[j][0], out.Edges[i].Points[j][1] = out.Edges[i].Points[j][1], out.Edges[i].Points[j][0]
			}
		}
		w, h := bboxOf(out)
		for i := range out.Nodes {
			out.Nodes[i].X = w - out.Nodes[i].X - out.Nodes[i].Width
		}
		for i := range out.Edges {
			for j := range out.Edges[i].Points {
				out.Edges[i].Points[j][0] = w - out.Edges[i].Points[j][0]
			}
		}
		out.Width, out.Height = w, h
	}
}
