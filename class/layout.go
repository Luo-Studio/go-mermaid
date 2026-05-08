package class

import (
	"github.com/luo-studio/go-mermaid/autog"
	"github.com/luo-studio/go-mermaid/displaylist"
	"github.com/luo-studio/go-mermaid/layoutopts"
)

// Layout positions a class diagram and returns a DisplayList.
func Layout(d *Diagram, opts layoutopts.Options) *displaylist.DisplayList {
	if d == nil || len(d.Classes) == 0 {
		return &displaylist.DisplayList{}
	}
	measurer := opts.ResolveMeasurer()
	measure := measurer.Measure

	classSize := map[string]struct {
		W, H float64
	}{}
	classByID := map[string]Class{}

	for _, c := range d.Classes {
		classByID[c.ID] = c
		labelW, labelH := measure(c.Label, displaylist.RoleClassBox)
		w := labelW + 24
		h := labelH + 8
		if c.Annotation != "" {
			aw, ah := measure(c.Annotation, displaylist.RoleClassAnnotation)
			if aw+24 > w {
				w = aw + 24
			}
			h += ah + 4
		}
		bodyH := 0.0
		for _, m := range c.Members {
			line := formatMember(m)
			lw, lh := measure(line, displaylist.RoleClassMember)
			if lw+24 > w {
				w = lw + 24
			}
			bodyH += lh + 2
		}
		if bodyH > 0 {
			bodyH += 8
		}
		h += bodyH
		classSize[c.ID] = struct{ W, H float64 }{w, h}
	}

	var nodes []autog.Node
	for _, c := range d.Classes {
		s := classSize[c.ID]
		nodes = append(nodes, autog.Node{ID: c.ID, Width: s.W, Height: s.H})
	}
	var edges []autog.Edge
	for _, r := range d.Relationships {
		edges = append(edges, autog.Edge{FromID: r.From, ToID: r.To})
	}

	dir := autog.DirectionTB

	if len(d.Namespaces) == 0 {
		out, err := autog.Layout(autog.Input{
			Nodes: nodes, Edges: edges, Direction: dir,
			NodeSpacing: opts.NodeSpacing, LayerSpacing: opts.LayerSpacing,
		})
		if err != nil {
			return &displaylist.DisplayList{}
		}
		return assemble(out.Nodes, out.Edges, nil, out.Width, out.Height, classByID, d.Relationships, measure)
	}

	var clusters []*autog.Cluster
	for _, ns := range d.Namespaces {
		clusters = append(clusters, &autog.Cluster{
			ID:      "ns_" + ns.Name,
			Title:   ns.Name,
			NodeIDs: ns.ClassIDs,
		})
	}
	out, err := autog.LayoutClusters(autog.ClusterInput{
		Nodes: nodes, Edges: edges, Direction: dir,
		NodeSpacing: opts.NodeSpacing, LayerSpacing: opts.LayerSpacing,
		Clusters: clusters,
	})
	if err != nil {
		return &displaylist.DisplayList{}
	}
	dl := assemble(out.Nodes, out.Edges, out.ClusterRects, out.Width, out.Height, classByID, d.Relationships, measure)
	return dl
}

func assemble(nodes []autog.Node, edges []autog.Edge, rects []autog.ClusterRect, width, height float64, classByID map[string]Class, rels []Relationship, measure func(string, displaylist.Role) (float64, float64)) *displaylist.DisplayList {
	dl := &displaylist.DisplayList{Width: width, Height: height}
	emitClusterRects(dl, rects)
	for _, n := range nodes {
		c, ok := classByID[n.ID]
		if !ok {
			continue
		}
		emitClass(dl, c, displaylist.Rect{X: n.X, Y: n.Y, W: n.Width, H: n.Height}, measure)
	}
	for _, e := range edges {
		r := findRelationship(rels, e.FromID, e.ToID)
		emitRelationship(dl, e, r)
	}
	return dl
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

func emitClass(dl *displaylist.DisplayList, c Class, b displaylist.Rect, measure func(string, displaylist.Role) (float64, float64)) {
	dl.Items = append(dl.Items, displaylist.Shape{
		Kind: displaylist.ShapeKindRect,
		BBox: b,
		Role: displaylist.RoleClassBox,
	})
	cursorY := b.Y + 4
	if c.Annotation != "" {
		_, ah := measure(c.Annotation, displaylist.RoleClassAnnotation)
		dl.Items = append(dl.Items, displaylist.Text{
			Pos:    displaylist.Point{X: b.X + b.W/2, Y: cursorY + ah/2},
			Lines:  []string{c.Annotation},
			Align:  displaylist.AlignCenter,
			VAlign: displaylist.VAlignMiddle,
			Role:   displaylist.RoleClassAnnotation,
		})
		cursorY += ah + 2
	}
	_, nh := measure(c.Label, displaylist.RoleClassBox)
	dl.Items = append(dl.Items, displaylist.Text{
		Pos:    displaylist.Point{X: b.X + b.W/2, Y: cursorY + nh/2},
		Lines:  []string{c.Label},
		Align:  displaylist.AlignCenter,
		VAlign: displaylist.VAlignMiddle,
		Role:   displaylist.RoleClassBox,
	})
	cursorY += nh + 4
	if len(c.Members) > 0 {
		dl.Items = append(dl.Items, displaylist.Edge{
			Points:    []displaylist.Point{{X: b.X, Y: cursorY}, {X: b.X + b.W, Y: cursorY}},
			LineStyle: displaylist.LineStyleSolid,
			Role:      displaylist.RoleClassBox,
		})
		cursorY += 4
		for _, m := range c.Members {
			line := formatMember(m)
			_, lh := measure(line, displaylist.RoleClassMember)
			dl.Items = append(dl.Items, displaylist.Text{
				Pos:    displaylist.Point{X: b.X + 6, Y: cursorY + lh/2},
				Lines:  []string{line},
				Align:  displaylist.AlignLeft,
				VAlign: displaylist.VAlignMiddle,
				Role:   displaylist.RoleClassMember,
			})
			cursorY += lh + 2
		}
	}
}

func emitRelationship(dl *displaylist.DisplayList, e autog.Edge, r Relationship) {
	pts := make([]displaylist.Point, len(e.Points))
	for i, p := range e.Points {
		pts[i] = displaylist.Point{X: p[0], Y: p[1]}
	}
	style := displaylist.LineStyleSolid
	if r.Dashed {
		style = displaylist.LineStyleDashed
	}
	startMarker, endMarker := relationshipMarkers(r.Kind)
	dl.Items = append(dl.Items, displaylist.Edge{
		Points:     pts,
		LineStyle:  style,
		ArrowStart: startMarker,
		ArrowEnd:   endMarker,
		Role:       displaylist.RoleEdge,
	})
	if r.Label != "" && len(pts) > 0 {
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
			Lines:  []string{r.Label},
			Align:  displaylist.AlignCenter,
			VAlign: displaylist.VAlignBottom,
			Role:   displaylist.RoleEdgeLabel,
		})
	}
	if r.FromCard != "" && len(pts) > 0 {
		dl.Items = append(dl.Items, displaylist.Text{
			Pos:    displaylist.Point{X: pts[0].X + 4, Y: pts[0].Y - 2},
			Lines:  []string{r.FromCard},
			Align:  displaylist.AlignLeft,
			VAlign: displaylist.VAlignBottom,
			Role:   displaylist.RoleEdgeLabel,
		})
	}
	if r.ToCard != "" && len(pts) > 0 {
		end := pts[len(pts)-1]
		dl.Items = append(dl.Items, displaylist.Text{
			Pos:    displaylist.Point{X: end.X - 4, Y: end.Y - 2},
			Lines:  []string{r.ToCard},
			Align:  displaylist.AlignRight,
			VAlign: displaylist.VAlignBottom,
			Role:   displaylist.RoleEdgeLabel,
		})
	}
}

func relationshipMarkers(k RelationKind) (start, end displaylist.MarkerKind) {
	switch k {
	case RelInheritance, RelRealization:
		return displaylist.MarkerTriangleOpen, displaylist.MarkerNone
	case RelComposition:
		return displaylist.MarkerDiamondFilled, displaylist.MarkerNone
	case RelAggregation:
		return displaylist.MarkerDiamondOpen, displaylist.MarkerNone
	case RelDependency, RelAssociation:
		return displaylist.MarkerNone, displaylist.MarkerArrow
	}
	return displaylist.MarkerNone, displaylist.MarkerNone
}

func findRelationship(rels []Relationship, from, to string) Relationship {
	for _, r := range rels {
		if r.From == from && r.To == to {
			return r
		}
	}
	return Relationship{}
}

func formatMember(m Member) string {
	v := ""
	switch m.Visibility {
	case VisPublic:
		v = "+"
	case VisPrivate:
		v = "-"
	case VisProtected:
		v = "#"
	case VisPackage:
		v = "~"
	}
	suffix := ""
	if m.IsStatic {
		suffix = " $"
	}
	if m.IsAbstract {
		suffix += " *"
	}
	if m.IsMethod {
		out := v + m.Name + m.Args
		if m.Type != "" {
			out += " : " + m.Type
		}
		return out + suffix
	}
	out := v + m.Name
	if m.Type != "" {
		out += " : " + m.Type
	}
	return out + suffix
}
