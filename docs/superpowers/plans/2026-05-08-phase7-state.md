# go-mermaid Phase 7 — State Diagrams

> **For agentic workers:** REQUIRED SUB-SKILL: superpowers:subagent-driven-development.

**Goal:** Add state-diagram support: parser, autog-based layout with composite states (treated as clusters), DisplayList emit with state-specific shape kinds for pseudostates (filled bullet for `[*]` start; bullseye for `[*]` end), PDF emitter extensions.

**Architecture:** The `state` package owns parser + AST + layout. `[*]` is special — it is a pseudostate, not a regular state. Composite states `state X { ... }` are clusters using the autog cluster engine from Phase 3. State diagrams accept `stateDiagram` or `stateDiagram-v2` headers; both use the same parser.

**Depends on:** Phase 1, 2, 3.

---

## Spec Reference

Spec section "State Diagram Layout".

## File Structure

```
go-mermaid/
├── state/
│   ├── ast.go
│   ├── parser.go
│   ├── parser_test.go
│   ├── layout.go
│   ├── layout_test.go
│   └── golden_test.go
├── pdf/
│   ├── style.go              # MODIFIED: state roles
│   └── draw_shape.go         # MODIFIED: pseudostate kinds (filled bullet, bullseye)
├── displaylist/
│   └── kind.go               # MODIFIED: ShapeKindStateBullet, ShapeKindStateBullseye
├── mermaid.go                # MODIFIED: dispatch typeState
└── testdata/state/{parse,layout}/...
```

## Tasks

### Task 1: AST + new ShapeKinds

**Files:** Create `state/ast.go`, modify `displaylist/kind.go`.

```go
// state/ast.go
package state

type State struct {
	ID     string  // "[*]" for pseudostates
	Label  string
	IsStart bool   // true if this is a `[*]` used as a transition source
	IsEnd   bool   // true if this is a `[*]` used as a transition target
	IsComposite bool
	Children []string // sub-state IDs, populated for composite states
	Note    string  // optional note attached to the state
}

type Transition struct {
	From  string
	To    string
	Label string
	// Trigger/Guard/Action sub-fields if we ever parse them; mermaid
	// keeps it simple — Label is the entire `: trigger / action` line.
}

type Diagram struct {
	States      []State
	Transitions []Transition
	Composites  []*Composite // composite states (becomes clusters in layout)
}

type Composite struct {
	ID       string
	Label    string
	StateIDs []string
	Children []*Composite // nested composites
}
```

Add to `displaylist/kind.go`:
```go
const (
	// ... existing kinds ...
	ShapeKindStateBullet   ShapeKind = "stateBullet"   // filled circle, ~10px
	ShapeKindStateBullseye ShapeKind = "stateBullseye" // ring around bullet
)
```

Add to `displaylist/role.go` if missing: `RoleStateBox`, `RoleStateComposite`, `RolePseudostateStart`, `RolePseudostateEnd` (already in spec; add if Phase 1 didn't include).

- [ ] Test + impl + commit: `git commit -m "state: AST types + pseudostate ShapeKinds"`

---

### Task 2: Parser

Implementation outline:
- First line must match `stateDiagram` or `stateDiagram-v2`.
- Two main statement kinds:
  1. **Transition:** `A --> B : label` where A and B can be `[*]`. Whether `[*]` is start or end depends on its position (left = start, right = end).
  2. **Composite state:** `state CompositeName { ... }` block — recurse into the body.
  3. **State annotation:** `state X { ... }` (composite) or `state "Display Label" as X` (alias).
  4. **Note:** `note left of X : text` / `note right of X : text` — attach to the State.
  5. Comments `%%` ignored.

Pseudostate handling: when a transition has `[*]` on the from side, generate a synthetic State with ID `__start__` (unique per composite scope) and IsStart=true. Similarly for `[*]` on the to side: ID `__end__` and IsEnd=true.

- [ ] Test + impl + commit: `git commit -m "state: parser (transitions, composites, notes, pseudostates)"`

---

### Task 3: Layout

```go
func Layout(d *Diagram, opts mermaid.LayoutOptions) *displaylist.DisplayList {
	measure := opts.Measurer.Measure

	// Size each state.
	nodes := []autog.Node{}
	for _, s := range d.States {
		if s.IsStart || s.IsEnd {
			// Pseudostates: small fixed size.
			nodes = append(nodes, autog.Node{ID: s.ID, Width: 16, Height: 16})
			continue
		}
		w, h := measure(s.Label, displaylist.RoleStateBox)
		nodes = append(nodes, autog.Node{ID: s.ID, Width: w + 24, Height: h + 12})
	}
	edges := []autog.Edge{}
	for _, t := range d.Transitions {
		edges = append(edges, autog.Edge{FromID: t.From, ToID: t.To})
	}
	clusters := convertComposites(d.Composites)
	out, err := autog.LayoutClusters(autog.ClusterInput{
		Nodes: nodes, Edges: edges, Clusters: clusters,
		Direction: autog.DirectionTB,
		NodeSpacing: opts.NodeSpacing, LayerSpacing: opts.LayerSpacing,
	})
	if err != nil { return &displaylist.DisplayList{} }

	dl := &displaylist.DisplayList{Width: out.Width, Height: out.Height}
	emitClusters(dl, out.ClusterRects, displaylist.RoleStateComposite)
	stateByID := map[string]State{}
	for _, s := range d.States { stateByID[s.ID] = s }
	for _, n := range out.Nodes {
		s := stateByID[n.ID]
		bbox := displaylist.Rect{X: n.X, Y: n.Y, W: n.Width, H: n.Height}
		switch {
		case s.IsStart:
			dl.Items = append(dl.Items, displaylist.Shape{
				Kind: displaylist.ShapeKindStateBullet, BBox: bbox,
				Role: displaylist.RolePseudostateStart,
			})
		case s.IsEnd:
			dl.Items = append(dl.Items, displaylist.Shape{
				Kind: displaylist.ShapeKindStateBullseye, BBox: bbox,
				Role: displaylist.RolePseudostateEnd,
			})
		default:
			dl.Items = append(dl.Items, displaylist.Shape{
				Kind: displaylist.ShapeKindRound, BBox: bbox,
				Role: displaylist.RoleStateBox,
			})
			dl.Items = append(dl.Items, displaylist.Text{
				Pos: displaylist.Point{X: bbox.X + bbox.W/2, Y: bbox.Y + bbox.H/2},
				Lines: []string{s.Label},
				Align: displaylist.AlignCenter, VAlign: displaylist.VAlignMiddle,
				Role: displaylist.RoleStateBox,
			})
		}
	}
	for _, e := range out.Edges {
		t := findTransition(d, e.FromID, e.ToID)
		pts := make([]displaylist.Point, len(e.Points))
		for i, p := range e.Points { pts[i] = displaylist.Point{X: p[0], Y: p[1]} }
		dl.Items = append(dl.Items, displaylist.Edge{
			Points: pts, LineStyle: displaylist.LineStyleSolid,
			ArrowEnd: displaylist.MarkerArrow,
			Role: displaylist.RoleEdge,
		})
		if t.Label != "" {
			dl.Items = append(dl.Items, displaylist.Text{
				Pos: pts[len(pts)/2], Lines: []string{t.Label},
				Align: displaylist.AlignCenter, VAlign: displaylist.VAlignBottom,
				Role: displaylist.RoleEdgeLabel,
			})
		}
	}
	return dl
}
```

- [ ] Test + impl + commit: `git commit -m "state: layout (autog + composites + pseudostates)"`

---

### Task 4: PDF emitter — pseudostate shapes

**Files:** Modify `pdf/draw_shape.go` to add new ShapeKind cases.

```go
case displaylist.ShapeKindStateBullet:
	cx, cy := x+w/2, y+h/2
	r := w / 3
	pdf.SetFillColor(int(rs.StrokeR), int(rs.StrokeG), int(rs.StrokeB))
	pdf.Ellipse(cx, cy, r, r, 0, "F")
case displaylist.ShapeKindStateBullseye:
	cx, cy := x+w/2, y+h/2
	rOuter := w / 2.5
	rInner := w / 4
	pdf.Ellipse(cx, cy, rOuter, rOuter, 0, "D")
	pdf.SetFillColor(int(rs.StrokeR), int(rs.StrokeG), int(rs.StrokeB))
	pdf.Ellipse(cx, cy, rInner, rInner, 0, "F")
```

- [ ] Commit: `git commit -m "mermaidpdf: pseudostate bullet + bullseye"`

---

### Task 5: Wire + tests + smoke

- Wire `case typeState:` in `mermaid.go`.
- Golden tests for parse + layout. Property tests.
- Phase 7 smoke test:

```go
func TestPhase7Smoke(t *testing.T) {
	src := `stateDiagram-v2
[*] --> Idle
Idle --> Running : start
Running --> Idle : stop
Running --> [*]
state Running {
  Working --> Waiting : block
  Waiting --> Working : unblock
}
`
	dl, err := ParseAndLayout(src, LayoutOptions{})
	if err != nil { t.Fatal(err) }
	if dl.Width <= 0 { t.Fatal("empty bbox") }
	// Expect at least one bullet (start) and one bullseye (end).
	bulletCount, bullseyeCount := 0, 0
	for _, it := range dl.Items {
		if s, ok := it.(displaylist.Shape); ok {
			if s.Kind == displaylist.ShapeKindStateBullet { bulletCount++ }
			if s.Kind == displaylist.ShapeKindStateBullseye { bullseyeCount++ }
		}
	}
	if bulletCount < 1 || bullseyeCount < 1 {
		t.Fatalf("expected ≥1 bullet and ≥1 bullseye, got %d/%d", bulletCount, bullseyeCount)
	}
}
```

- [ ] Commits: `mermaid: dispatch state`, `state: golden + property tests`, `mermaid: phase 7 smoke`.

---

## Open Questions / Risks

- **Concurrent regions** (`--` divider inside a composite state): mermaid-js supports parallel regions inside one composite. Phase 7 v1 ignores; treat the divider as a comment. Add later if real samples need it.
- **Choice/fork/join nodes** (`<<choice>>`, `<<fork>>`, `<<join>>`): deferred. Treat as regular states until users complain.
- **History pseudostates** (`[H]`, `[H*]`): deferred.
