# go-mermaid Phase 5 — Class Diagrams

> **For agentic workers:** REQUIRED SUB-SKILL: superpowers:subagent-driven-development.

**Goal:** Add class-diagram support: parser, autog-based layout, DisplayList emit with class-specific roles, PDF emitter extensions for relationship markers (filled diamond, hollow diamond, hollow triangle, etc.).

**Architecture:** The `class` package owns parser + AST + layout. Parser handles `class X { members }`, inline attribute lines (`X : +method() void`), relationships (`A <|-- B`, `A "1" --> "*" B : has`), annotations (`<<interface>>`), and `namespace X { ... }` blocks (treated as clusters via the autog cluster machinery from Phase 3). Layout runs each class as one autog node sized to fit its name + members. Relationship markers are rendered by the PDF emitter via new MarkerKinds.

**Depends on:** Phase 1, 2, 3 (cluster engine — needed for `namespace`).

---

## Spec Reference

Spec section "Class Feature Coverage".

## File Structure

```
go-mermaid/
├── class/
│   ├── ast.go
│   ├── parser.go
│   ├── parser_test.go
│   ├── layout.go
│   ├── layout_test.go
│   └── golden_test.go
├── pdf/
│   ├── style.go              # MODIFIED: class roles
│   └── draw_marker.go        # MODIFIED/NEW: triangleOpen, diamondFilled, diamondOpen
├── mermaid.go                # MODIFIED: dispatch typeClass
└── testdata/class/{parse,layout}/...
```

## Tasks

### Task 1: AST

```go
package class

type Visibility int
const (
	VisDefault Visibility = iota // mermaid default = public
	VisPublic                    // +
	VisPrivate                   // -
	VisProtected                 // #
	VisPackage                   // ~
)

type RelationKind int
const (
	RelInheritance RelationKind = iota // <|--
	RelComposition                      // *--
	RelAggregation                      // o--
	RelAssociation                      // -->
	RelDependency                       // ..>
	RelRealization                      // ..|>
	RelLink                             // -- (plain line)
)

type Member struct {
	Visibility Visibility
	Name       string
	Type       string  // for attrs: type; for methods: return type
	Args       string  // for methods: "(arg1: T1, arg2: T2)"
	IsMethod   bool
	IsStatic   bool
	IsAbstract bool
}

type Class struct {
	ID         string
	Label      string  // mermaid: classes can have a different display label via "class X[Label]"
	Annotation string  // <<interface>>, <<abstract>>, ...
	Members    []Member
	Namespace  string  // empty if top-level
}

type Relationship struct {
	From     string
	To       string
	Kind     RelationKind
	Label    string
	FromCard string  // "1", "*", "0..1", etc; quoted in source
	ToCard   string
	// Whether the line is dashed (..-- or ..-->) vs solid (-- or -->).
	Dashed   bool
}

type Namespace struct {
	Name     string
	ClassIDs []string
}

type Diagram struct {
	Classes       []Class
	Relationships []Relationship
	Namespaces    []Namespace
}
```

- [ ] Test, implement, commit: `git commit -m "class: AST types"`

---

### Task 2: Parser

Implementation outline:
- First line `classDiagram`.
- Track current namespace via stack (push on `namespace X {`, pop on `}`).
- Each line is one of:
  - `class X { ... }` (multi-line block — read until matching `}`)
  - `class X : +member` (inline attribute syntax)
  - `class X` (declare only)
  - Relationship line (parsed via the relationship regex below)
  - `<<stereotype>>` after a class name to set Annotation

Relationship regex (try forms in order):
```
^(\w+)(?:\s+"([^"]+)")?\s+(<\|--|<\|\.\.|\*--|--\*|o--|--o|-->|\.\.>|\.\.\|>|--)\s+(?:"([^"]+)"\s+)?(\w+)(?:\s*:\s*(.*))?$
```

Capture groups: from, fromCard, arrow, toCard, to, label.

Map arrow to `RelationKind` and `Dashed`:
- `<|--` → Inheritance, dashed=false
- `<|..` → Inheritance, dashed=true
- `*--` → Composition (always solid; reverse `--*` swaps from/to)
- `o--` → Aggregation
- `--o` → Aggregation, swapped
- `-->` → Association
- `..>` → Dependency, dashed=true
- `..|>` → Realization, dashed=true
- `--` → Link, no arrow

Member parsing inside `class X { ... }`:
- Skip blank lines.
- Each non-blank line is one member: `[+|-|#|~]?(static)?(abstract)?\s*(name)\s*(\([^)]*\))?\s*(:\s*type)?`
- Detect method by presence of parentheses.

- [ ] Tests + impl + commit: `git commit -m "class: parser (classes, members, relationships, namespaces)"`

---

### Task 3: Layout

```go
func Layout(d *Diagram, opts mermaid.LayoutOptions) *displaylist.DisplayList {
	measure := opts.Measurer.Measure
	// 1. Compute each class's bbox: title bar height + annotation
	// height (if any) + member-list height. Width = max(name, members).
	classSize := map[string]struct{ W, H float64 }{}
	for _, c := range d.Classes {
		nameW, nameH := measure(c.Label, displaylist.RoleClassBox)
		w := nameW + 24 // padding
		h := nameH + 8  // title bar height
		if c.Annotation != "" {
			aw, ah := measure(c.Annotation, displaylist.RoleClassAnnotation)
			if aw+24 > w { w = aw + 24 }
			h += ah + 4
		}
		// Body: each member is one line.
		bodyH := 0.0
		for _, m := range c.Members {
			line := formatMember(m)
			lw, lh := measure(line, displaylist.RoleClassMember)
			if lw+24 > w { w = lw + 24 }
			bodyH += lh + 2
		}
		if bodyH > 0 { bodyH += 8 } // top/bottom padding
		h += bodyH
		classSize[c.ID] = struct{ W, H float64 }{w, h}
	}

	// 2. Build cluster input from namespaces.
	nodes := []autog.Node{}
	for _, c := range d.Classes {
		s := classSize[c.ID]
		nodes = append(nodes, autog.Node{ID: c.ID, Width: s.W, Height: s.H})
	}
	edges := []autog.Edge{}
	for _, r := range d.Relationships {
		edges = append(edges, autog.Edge{FromID: r.From, ToID: r.To})
	}
	clusters := []*autog.Cluster{}
	for _, ns := range d.Namespaces {
		clusters = append(clusters, &autog.Cluster{
			ID: "ns_" + ns.Name, Title: ns.Name, NodeIDs: ns.ClassIDs,
		})
	}
	out, err := autog.LayoutClusters(autog.ClusterInput{
		Nodes: nodes, Edges: edges, Clusters: clusters,
		NodeSpacing: opts.NodeSpacing, LayerSpacing: opts.LayerSpacing,
		Direction: autog.DirectionTB,
	})
	if err != nil { return &displaylist.DisplayList{} }

	// 3. Emit DisplayList.
	dl := &displaylist.DisplayList{Width: out.Width, Height: out.Height}
	emitClusters(dl, out.ClusterRects, displaylist.RoleSubgraph) // namespaces
	classByID := map[string]Class{}
	for _, c := range d.Classes { classByID[c.ID] = c }
	for _, n := range out.Nodes {
		c := classByID[n.ID]
		emitClass(dl, c, displaylist.Rect{X: n.X, Y: n.Y, W: n.Width, H: n.Height}, measure)
	}
	for _, e := range out.Edges {
		r := findRelationship(d, e.FromID, e.ToID)
		emitRelationship(dl, e, r, measure)
	}
	return dl
}

// emitClass paints a 3-band rect: title bar (with optional annotation
// row), then a horizontal divider, then the member list.
func emitClass(dl *displaylist.DisplayList, c Class, b displaylist.Rect, measure func(string, displaylist.Role) (float64, float64)) {
	// Outer box.
	dl.Items = append(dl.Items, displaylist.Shape{Kind: displaylist.ShapeKindRect, BBox: b, Role: displaylist.RoleClassBox})
	// Annotation row (if any).
	cursorY := b.Y + 4
	if c.Annotation != "" {
		_, ah := measure(c.Annotation, displaylist.RoleClassAnnotation)
		dl.Items = append(dl.Items, displaylist.Text{
			Pos: displaylist.Point{X: b.X + b.W/2, Y: cursorY + ah/2},
			Lines: []string{c.Annotation}, Align: displaylist.AlignCenter, VAlign: displaylist.VAlignMiddle,
			Role: displaylist.RoleClassAnnotation,
		})
		cursorY += ah + 2
	}
	// Class name.
	_, nh := measure(c.Label, displaylist.RoleClassBox)
	dl.Items = append(dl.Items, displaylist.Text{
		Pos: displaylist.Point{X: b.X + b.W/2, Y: cursorY + nh/2},
		Lines: []string{c.Label}, Align: displaylist.AlignCenter, VAlign: displaylist.VAlignMiddle,
		Role: displaylist.RoleClassBox,
	})
	cursorY += nh + 4
	// Divider.
	dl.Items = append(dl.Items, displaylist.Edge{
		Points: []displaylist.Point{{X: b.X, Y: cursorY}, {X: b.X + b.W, Y: cursorY}},
		LineStyle: displaylist.LineStyleSolid, Role: displaylist.RoleClassBox,
	})
	cursorY += 4
	// Members.
	for _, m := range c.Members {
		line := formatMember(m)
		_, lh := measure(line, displaylist.RoleClassMember)
		dl.Items = append(dl.Items, displaylist.Text{
			Pos: displaylist.Point{X: b.X + 6, Y: cursorY + lh/2},
			Lines: []string{line}, Align: displaylist.AlignLeft, VAlign: displaylist.VAlignMiddle,
			Role: displaylist.RoleClassMember,
		})
		cursorY += lh + 2
	}
}

func emitRelationship(dl *displaylist.DisplayList, e autog.Edge, r Relationship, measure func(string, displaylist.Role) (float64, float64)) {
	pts := make([]displaylist.Point, len(e.Points))
	for i, p := range e.Points { pts[i] = displaylist.Point{X: p[0], Y: p[1]} }
	style := displaylist.LineStyleSolid
	if r.Dashed { style = displaylist.LineStyleDashed }
	startMarker, endMarker := relationshipMarkers(r.Kind)
	dl.Items = append(dl.Items, displaylist.Edge{
		Points: pts, LineStyle: style,
		ArrowStart: startMarker, ArrowEnd: endMarker,
		Role: displaylist.RoleEdge,
	})
	if r.Label != "" {
		mid := pts[len(pts)/2]
		dl.Items = append(dl.Items, displaylist.Text{
			Pos: mid, Lines: []string{r.Label},
			Align: displaylist.AlignCenter, VAlign: displaylist.VAlignBottom,
			Role: displaylist.RoleEdgeLabel,
		})
	}
	// Cardinality: small text near each endpoint.
	if r.FromCard != "" {
		dl.Items = append(dl.Items, displaylist.Text{
			Pos: pts[0], Lines: []string{r.FromCard},
			Role: displaylist.RoleEdgeLabel,
		})
	}
	if r.ToCard != "" {
		dl.Items = append(dl.Items, displaylist.Text{
			Pos: pts[len(pts)-1], Lines: []string{r.ToCard},
			Role: displaylist.RoleEdgeLabel,
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

func formatMember(m Member) string {
	v := ""
	switch m.Visibility {
	case VisPublic: v = "+"
	case VisPrivate: v = "-"
	case VisProtected: v = "#"
	case VisPackage: v = "~"
	}
	if m.IsMethod {
		return v + m.Name + m.Args + ifs(m.Type != "", " : "+m.Type, "")
	}
	return v + m.Name + ifs(m.Type != "", " : "+m.Type, "")
}
func ifs(c bool, t, f string) string { if c { return t }; return f }
```

- [ ] Tests + impl + commit: `git commit -m "class: layout via autog + DisplayList emit"`

---

### Task 4: PDF emitter — diamond/triangle markers

**Files:** Modify `pdf/draw_edge.go` (extend `drawArrow` switch).

```go
case displaylist.MarkerTriangleOpen:
	bx := tx1 - ux*size
	by := ty1 - uy*size
	pdf.Polygon([]fpdf.PointType{
		{X: tx1, Y: ty1},
		{X: bx + px*size*0.6, Y: by + py*size*0.6},
		{X: bx - px*size*0.6, Y: by - py*size*0.6},
	}, "D")
case displaylist.MarkerDiamondFilled, displaylist.MarkerDiamondOpen:
	bx := tx1 - ux*size*1.6
	by := ty1 - uy*size*1.6
	cx := tx1 - ux*size*0.8
	cy := ty1 - uy*size*0.8
	pdf.Polygon([]fpdf.PointType{
		{X: tx1, Y: ty1},
		{X: cx + px*size*0.5, Y: cy + py*size*0.5},
		{X: bx, Y: by},
		{X: cx - px*size*0.5, Y: cy - py*size*0.5},
	}, ifs(kind == displaylist.MarkerDiamondFilled, "F", "D"))
```

- [ ] Commit — `git commit -m "mermaidpdf: triangle and diamond markers"`

---

### Task 5: Wire + tests + smoke

- Wire `case typeClass:` in `mermaid.go`.
- Golden tests + property tests as in Phase 2/4.
- Phase 5 smoke test exercising inheritance, composition, aggregation, dependency.

- [ ] Commits: `mermaid: dispatch class`, `class: golden + property tests`, `mermaid: phase 5 smoke`.

---

## Open Questions / Risks

- **Generics `Class~T~`**: deferred (mermaigo also defers).
- **Class label vs ID**: Mermaid allows `class X[Display Label]`. Treat ID as the syntactic identifier and Label as display text; default Label = ID.
- **Cardinality positioning**: small offset from endpoint along edge direction would look better than placing at the exact endpoint. Tweak after visual review.
