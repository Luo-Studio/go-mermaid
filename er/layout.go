package er

import (
	"github.com/luo-studio/go-mermaid/autog"
	"github.com/luo-studio/go-mermaid/displaylist"
	"github.com/luo-studio/go-mermaid/internal/textutil"
	"github.com/luo-studio/go-mermaid/layoutopts"
)

// Layout positions an ER diagram and returns a DisplayList.
func Layout(d *Diagram, opts layoutopts.Options) *displaylist.DisplayList {
	if d == nil || len(d.Entities) == 0 {
		return &displaylist.DisplayList{}
	}
	measurer := opts.ResolveMeasurer()
	measure := measurer.Measure

	// Sizing is measurer-driven so it scales with the caller's font.
	// Padding is small in DisplayList units (typically mm) — the
	// Phase 6 placeholders of 22/16/12 were ~10x too large for an
	// fpdf doc rendered in mm with ~10pt body text.
	const (
		titlePadV = 4.0  // top/bottom padding inside the title bar
		rowPadV   = 1.0  // gap between attribute rows
		boxPadH   = 6.0  // horizontal text padding
		bottomPad = 4.0  // padding below the last attribute
	)

	_, titleLineH := measure("X", displaylist.RoleEntityBox)
	_, attrLineH := measure("X", displaylist.RoleEntityAttribute)

	var nodes []autog.Node
	entityByID := map[string]*Entity{}
	for _, e := range d.Entities {
		entityByID[e.ID] = e
		nameW, _ := measure(e.ID, displaylist.RoleEntityBox)
		w := nameW + boxPadH*2
		for _, a := range e.Attributes {
			line := formatAttribute(a)
			lw, _ := measure(line, displaylist.RoleEntityAttribute)
			if lw+boxPadH*2 > w {
				w = lw + boxPadH*2
			}
		}
		titleBarH := titleLineH + titlePadV*2
		bodyH := 0.0
		if n := len(e.Attributes); n > 0 {
			bodyH = float64(n)*(attrLineH+rowPadV) + bottomPad
		}
		h := titleBarH + bodyH
		nodes = append(nodes, autog.Node{ID: e.ID, Width: w, Height: h})
	}

	var edges []autog.Edge
	for _, r := range d.Relationships {
		edges = append(edges, autog.Edge{FromID: r.Left, ToID: r.Right})
	}

	out, err := autog.Layout(autog.Input{
		Nodes:        nodes,
		Edges:        edges,
		Direction:    autog.DirectionLR,
		NodeSpacing:  opts.NodeSpacing,
		LayerSpacing: opts.LayerSpacing,
	})
	if err != nil {
		return &displaylist.DisplayList{}
	}

	dl := &displaylist.DisplayList{Width: out.Width, Height: out.Height}
	titleBarH := titleLineH + titlePadV*2
	for _, n := range out.Nodes {
		emitEntity(dl, *entityByID[n.ID],
			displaylist.Rect{X: n.X, Y: n.Y, W: n.Width, H: n.Height},
			titleBarH, attrLineH+rowPadV)
	}
	for _, e := range out.Edges {
		r := findRelationship(d, e.FromID, e.ToID)
		emitRelationship(dl, e, r)
	}
	return dl
}

func emitEntity(dl *displaylist.DisplayList, e Entity, b displaylist.Rect, titleBarH, rowAdvance float64) {
	dl.Items = append(dl.Items, displaylist.Shape{
		Kind: displaylist.ShapeKindRect,
		BBox: b,
		Role: displaylist.RoleEntityBox,
	})
	dl.Items = append(dl.Items, displaylist.Text{
		Pos:    displaylist.Point{X: b.X + b.W/2, Y: b.Y + titleBarH/2},
		Lines:  textutil.SplitLabelLines(e.ID),
		Align:  displaylist.AlignCenter,
		VAlign: displaylist.VAlignMiddle,
		Role:   displaylist.RoleEntityBox,
	})
	if len(e.Attributes) > 0 {
		dl.Items = append(dl.Items, displaylist.Edge{
			Points:    []displaylist.Point{{X: b.X, Y: b.Y + titleBarH}, {X: b.X + b.W, Y: b.Y + titleBarH}},
			LineStyle: displaylist.LineStyleSolid,
			Role:      displaylist.RoleEntityBox,
		})
	}
	cy := b.Y + titleBarH + rowAdvance/2
	for _, a := range e.Attributes {
		dl.Items = append(dl.Items, displaylist.Text{
			Pos:    displaylist.Point{X: b.X + 6, Y: cy},
			Lines:  textutil.SplitLabelLines(formatAttribute(a)),
			Align:  displaylist.AlignLeft,
			VAlign: displaylist.VAlignMiddle,
			Role:   displaylist.RoleEntityAttribute,
		})
		cy += rowAdvance
	}
}

func emitRelationship(dl *displaylist.DisplayList, e autog.Edge, r *Relationship) {
	pts := make([]displaylist.Point, len(e.Points))
	for i, p := range e.Points {
		pts[i] = displaylist.Point{X: p[0], Y: p[1]}
	}
	style := displaylist.LineStyleSolid
	if r != nil && !r.Identifying {
		style = displaylist.LineStyleDashed
	}
	var leftCard, rightCard displaylist.MarkerKind
	if r != nil {
		leftCard = cardMarker(r.LeftCardinality)
		rightCard = cardMarker(r.RightCardinality)
	}
	dl.Items = append(dl.Items, displaylist.Edge{
		Points:     pts,
		LineStyle:  style,
		ArrowStart: leftCard,
		ArrowEnd:   rightCard,
		Role:       displaylist.RoleEdge,
	})
	if r != nil && r.Label != "" && len(pts) > 0 {
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
			Lines:  textutil.SplitLabelLines(r.Label),
			Align:  displaylist.AlignCenter,
			VAlign: displaylist.VAlignBottom,
			Role:   displaylist.RoleEdgeLabel,
		})
	}
}

func cardMarker(c Cardinality) displaylist.MarkerKind {
	switch c {
	case CardExactlyOne:
		return displaylist.MarkerCardinalityOne
	case CardZeroOrOne:
		return displaylist.MarkerCardinalityZeroOrOne
	case CardZeroOrMore:
		return displaylist.MarkerCardinalityZeroOrMore
	case CardOneOrMore:
		return displaylist.MarkerCardinalityOneOrMore
	}
	return displaylist.MarkerNone
}

func formatAttribute(a Attribute) string {
	s := a.Type + " " + a.Name
	switch a.Key {
	case KeyPrimary:
		s += " PK"
	case KeyForeign:
		s += " FK"
	case KeyUnique:
		s += " UK"
	}
	if a.Comment != "" {
		s += `  "` + a.Comment + `"`
	}
	return s
}

func findRelationship(d *Diagram, from, to string) *Relationship {
	for _, r := range d.Relationships {
		if r.Left == from && r.Right == to {
			return r
		}
		if r.Left == to && r.Right == from {
			// edge direction may be reversed by autog; return mirrored
			return &Relationship{
				Left:             r.Right,
				Right:            r.Left,
				LeftCardinality:  r.RightCardinality,
				RightCardinality: r.LeftCardinality,
				Identifying:      r.Identifying,
				Label:            r.Label,
			}
		}
	}
	return nil
}
