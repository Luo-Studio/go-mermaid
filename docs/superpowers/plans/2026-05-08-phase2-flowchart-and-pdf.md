# go-mermaid Phase 2 — Flowchart parser + Layout + Minimal PDF Emitter

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Ship the first end-to-end useful version of go-mermaid: a flowchart parser, an autog-backed layout (no clusters yet — Phase 3), and a minimal `pdf` emitter that draws the result into an fpdf document. After this phase `mermaidpdf.DrawMermaid(pdf, "flowchart TD\n  A --> B", x, y, opts)` produces a real diagram.

**Architecture:** The `flowchart` package owns parser + AST + layout for `graph`/`flowchart` keyword diagrams. Parsing is a line-by-line regex approach (Mermaid's flowchart grammar is regular enough that a parser combinator is overkill). Layout walks the AST, computes node sizes via the configured `Measurer`, calls `autog.Layout`, and emits a `displaylist.DisplayList`. Subgraph blocks are *parsed* but flattened in this phase; cluster recursion lands in Phase 3. The new `pdf` package draws DisplayList items into an fpdf canvas with a role-keyed Style map.

**Tech Stack:** `github.com/luo-studio/go-mermaid/{displaylist,fontmetrics,fonts,autog}` (Phase 1), `github.com/luo-studio/go-mermaid/flowchart` (this phase), `codeberg.org/go-pdf/fpdf` (replaced via `github.com/luo-studio/fpdf` per platform convention).

**Depends on:** Phase 1 plan complete (spec branch up to commit that lands `mermaid.ParseAndLayout`/`Measurer`/`displaylist.*`/`autog.Layout`/`fonts`/`fontmetrics`).

---

## Spec Reference

- Spec: `docs/superpowers/specs/2026-05-08-go-mermaid-design.md` — sections "Flowchart Feature Coverage", "Public API Surface" (PDF emitter), "DisplayList Primitive Set".

## File Structure (Phase 2)

```
go-mermaid/
├── flowchart/
│   ├── ast.go                       # AST types: Diagram, Node, Edge, Subgraph, ClassDef, Direction
│   ├── parser.go                    # Parse(src) — regex-driven line parser
│   ├── parser_test.go               # parser unit tests + golden ASTs
│   ├── layout.go                    # Layout(ast, opts) — autog adapter + DisplayList emit
│   ├── layout_test.go               # layout golden tests
│   └── shapes.go                    # node-shape sizing (text+padding → bbox)
├── pdf/
│   ├── emit.go                      # DrawInto, DrawMermaid, EmbedOptions, Style, RoleStyle, EmbedDefaults
│   ├── draw_shape.go                # per-ShapeKind drawing (rect/round/diamond/...)
│   ├── draw_edge.go                 # polyline + arrow heads
│   ├── draw_text.go                 # text run with role-driven font selection
│   ├── emit_test.go                 # PDF integration tests
│   └── style.go                     # default Style + role lookup helper
├── mermaid.go                       # MODIFIED: dispatch typeFlowchart to flowchart.Parse + Layout
├── mermaid_test.go                  # MODIFIED: ErrNotImplemented now only for class/er/state/sequence
├── testdata/flowchart/
│   ├── parse/
│   │   ├── simple-tb.mmd
│   │   ├── simple-tb.golden.json
│   │   ├── all-shapes.mmd
│   │   ├── all-shapes.golden.json
│   │   ├── all-edges.mmd
│   │   ├── all-edges.golden.json
│   │   └── classdef.mmd / .golden.json
│   └── layout/
│       ├── simple-tb.mmd / .golden.json
│       └── all-shapes.mmd / .golden.json
└── testdata/pdf/
    └── simple-tb.expected-objects     # number of expected PDF objects
```

## Tasks

### Task 1: AST types

**Files:** Create `flowchart/ast.go`, `flowchart/ast_test.go`.

- [ ] **Step 1.1: Write the failing test**

```go
// flowchart/ast_test.go
package flowchart

import "testing"

func TestDiagramZeroValue(t *testing.T) {
	var d Diagram
	if d.Direction != DirectionTB {
		t.Fatalf("default direction should be TB, got %v", d.Direction)
	}
	if len(d.Nodes) != 0 || len(d.Edges) != 0 {
		t.Fatal("zero diagram should be empty")
	}
}

func TestDirectionString(t *testing.T) {
	cases := map[Direction]string{
		DirectionTB: "TB", DirectionBT: "BT", DirectionLR: "LR", DirectionRL: "RL",
	}
	for d, want := range cases {
		if d.String() != want {
			t.Errorf("Direction(%d).String() = %q, want %q", d, d.String(), want)
		}
	}
}
```

- [ ] **Step 1.2: Run, expect FAIL.** `go test ./flowchart/...`
- [ ] **Step 1.3: Implement `flowchart/ast.go`**

```go
// Package flowchart parses Mermaid `flowchart`/`graph` source into an
// AST and lays it out into a displaylist.DisplayList.
package flowchart

// Direction is the diagram's flow direction.
type Direction int

const (
	DirectionTB Direction = iota // top→bottom (also "TD" in source)
	DirectionBT                  // bottom→top
	DirectionLR                  // left→right
	DirectionRL                  // right→left
)

func (d Direction) String() string {
	switch d {
	case DirectionBT:
		return "BT"
	case DirectionLR:
		return "LR"
	case DirectionRL:
		return "RL"
	default:
		return "TB"
	}
}

// NodeShape mirrors mermaid's node-shape syntax. Maps to
// displaylist.ShapeKind in layout.
type NodeShape int

const (
	ShapeRect NodeShape = iota
	ShapeRound
	ShapeStadium
	ShapeDiamond
	ShapeCircle
	ShapeDoubleCircle
	ShapeSubroutine
	ShapeCylinder
	ShapeHexagon
	ShapeAsymmetric
	ShapeTrapezoid
	ShapeTrapezoidAlt
	ShapeParallelogram
	ShapeParallelogramAlt
	ShapeStateStart
	ShapeStateEnd
)

// EdgeStyle is the line style of an edge.
type EdgeStyle int

const (
	EdgeSolid EdgeStyle = iota
	EdgeDotted
	EdgeThick
)

// Node is a single node in the flowchart.
type Node struct {
	ID         string
	Label      string
	Shape      NodeShape
	ClassNames []string // from ::: shorthand or `class A foo`
}

// Edge is a connection between two nodes.
type Edge struct {
	From          string
	To            string
	Label         string
	Style         EdgeStyle
	ArrowStart    bool // true for bidirectional edges
	ArrowEnd      bool // false for non-arrow lines (`---`, `-.-`, `===`)
}

// Subgraph is a `subgraph X ... end` block. Children are node IDs and
// nested subgraphs.
type Subgraph struct {
	ID        string
	Label     string
	NodeIDs   []string
	Children  []*Subgraph
	Direction Direction // optional override; inherits parent if zero-value
}

// ClassDef is a `classDef name property:value;...` declaration. The
// property map is propagated to nodes that name this class via ::: or
// `class N name`.
type ClassDef struct {
	Name       string
	Properties map[string]string
}

// Diagram is the parsed flowchart.
type Diagram struct {
	Direction  Direction
	Nodes      []Node
	Edges      []Edge
	Subgraphs  []*Subgraph
	ClassDefs  []ClassDef
	NodeStyles map[string]map[string]string // node-ID → property → value (from inline `style A foo:bar`)
}
```

- [ ] **Step 1.4: Run, expect PASS.**
- [ ] **Step 1.5: Commit** — `git add flowchart/ast.go flowchart/ast_test.go && git commit -m "flowchart: AST types"`

---

### Task 2: Parser — preprocess + direction header

**Files:** Create `flowchart/parser.go`, extend `flowchart/parser_test.go`.

- [ ] **Step 2.1: Write the failing test**

```go
// flowchart/parser_test.go
package flowchart

import "testing"

func TestParseDirection(t *testing.T) {
	cases := map[string]Direction{
		"flowchart TB\n":      DirectionTB,
		"flowchart TD\n":      DirectionTB, // TD == TB
		"flowchart BT\n":      DirectionBT,
		"flowchart LR\n":      DirectionLR,
		"flowchart RL\n":      DirectionRL,
		"graph LR\n":          DirectionLR,
		"  graph    LR  \n":   DirectionLR,
		"%% comment\nflowchart RL\n": DirectionRL,
	}
	for src, want := range cases {
		t.Run(src, func(t *testing.T) {
			d, err := Parse(src)
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			if d.Direction != want {
				t.Errorf("got %v want %v", d.Direction, want)
			}
		})
	}
}

func TestParseRejectsUnknownHeader(t *testing.T) {
	if _, err := Parse("not-a-flowchart\nA --> B\n"); err == nil {
		t.Fatal("expected error for non-flowchart header")
	}
}

func TestParseEmpty(t *testing.T) {
	d, err := Parse("flowchart TB\n")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(d.Nodes) != 0 || len(d.Edges) != 0 {
		t.Fatal("empty flowchart should have no nodes or edges")
	}
}
```

- [ ] **Step 2.2: Run, expect FAIL** (Parse undefined).
- [ ] **Step 2.3: Implement `flowchart/parser.go`** (header-only initially):

```go
package flowchart

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	headerRE = regexp.MustCompile(`(?i)^(flowchart|graph)(?:\s+(TB|TD|BT|LR|RL))?\s*$`)
)

// Parse turns Mermaid flowchart source into a Diagram.
//
// The source must start with a `flowchart` or `graph` header line
// (after any leading blank lines or %% comments). Subsequent lines
// declare nodes, edges, subgraphs, classDefs, and styles.
func Parse(src string) (*Diagram, error) {
	lines := preprocess(src)
	if len(lines) == 0 {
		return nil, fmt.Errorf("flowchart: empty source")
	}
	d := &Diagram{
		Direction:  DirectionTB,
		NodeStyles: map[string]map[string]string{},
	}
	m := headerRE.FindStringSubmatch(lines[0])
	if m == nil {
		return nil, fmt.Errorf("flowchart: line 1: expected `flowchart` or `graph` header, got %q", lines[0])
	}
	switch strings.ToUpper(m[2]) {
	case "TB", "TD", "":
		d.Direction = DirectionTB
	case "BT":
		d.Direction = DirectionBT
	case "LR":
		d.Direction = DirectionLR
	case "RL":
		d.Direction = DirectionRL
	}
	// Body lines parsed in subsequent tasks.
	_ = lines[1:]
	return d, nil
}

// preprocess splits src by newlines and semicolons, trims each
// segment, and drops empty/comment lines. Comments start with %%.
func preprocess(src string) []string {
	var out []string
	for _, line := range strings.Split(src, "\n") {
		// Split on `;` so `A --> B; B --> C` becomes two lines.
		for _, frag := range strings.Split(line, ";") {
			s := strings.TrimSpace(frag)
			if s == "" || strings.HasPrefix(s, "%%") {
				continue
			}
			out = append(out, s)
		}
	}
	return out
}
```

- [ ] **Step 2.4: Run, expect PASS.**
- [ ] **Step 2.5: Commit** — `git commit -m "flowchart: parser preprocess + header"`

---

### Task 3: Parser — node + edge statements

**Files:** Modify `flowchart/parser.go`, extend `flowchart/parser_test.go`.

- [ ] **Step 3.1: Write the failing test**

```go
// flowchart/parser_test.go (append)
func TestParseEdges(t *testing.T) {
	d, err := Parse("flowchart TB\nA --> B\nB --- C\nC -.-> D\nD ==> E\n")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(d.Edges) != 4 {
		t.Fatalf("want 4 edges, got %d", len(d.Edges))
	}
	if d.Edges[0].From != "A" || d.Edges[0].To != "B" || !d.Edges[0].ArrowEnd {
		t.Errorf("edge 0: got %+v", d.Edges[0])
	}
	if d.Edges[1].ArrowEnd {
		t.Errorf("edge 1: --- should have no end arrow: %+v", d.Edges[1])
	}
	if d.Edges[2].Style != EdgeDotted {
		t.Errorf("edge 2: want dotted, got %v", d.Edges[2].Style)
	}
	if d.Edges[3].Style != EdgeThick {
		t.Errorf("edge 3: want thick, got %v", d.Edges[3].Style)
	}
}

func TestParseEdgeWithLabel(t *testing.T) {
	d, _ := Parse("flowchart TB\nA -- yes --> B\nA -->|no| C\n")
	if d.Edges[0].Label != "yes" {
		t.Errorf("edge 0 label: %q", d.Edges[0].Label)
	}
	if d.Edges[1].Label != "no" {
		t.Errorf("edge 1 label: %q", d.Edges[1].Label)
	}
}

func TestParseBidirectional(t *testing.T) {
	d, _ := Parse("flowchart TB\nA <--> B\n")
	e := d.Edges[0]
	if !e.ArrowStart || !e.ArrowEnd {
		t.Errorf("bidirectional should set both arrows: %+v", e)
	}
}

func TestParseShapes(t *testing.T) {
	src := `flowchart TB
A[Rect]
B(Round)
C([Stadium])
D{Diamond}
E((Circle))
F(((DoubleCircle)))
G[[Subroutine]]
H[(Cylinder)]
I{{Hexagon}}
J>Asymmetric]
K[/Trapezoid\]
L[\TrapezoidAlt/]
A --> B
B --> C
C --> D
`
	d, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	wantShapes := []NodeShape{
		ShapeRect, ShapeRound, ShapeStadium, ShapeDiamond,
		ShapeCircle, ShapeDoubleCircle, ShapeSubroutine, ShapeCylinder,
		ShapeHexagon, ShapeAsymmetric, ShapeTrapezoid, ShapeTrapezoidAlt,
	}
	if len(d.Nodes) != len(wantShapes) {
		t.Fatalf("node count: got %d want %d", len(d.Nodes), len(wantShapes))
	}
	byID := map[string]Node{}
	for _, n := range d.Nodes {
		byID[n.ID] = n
	}
	ids := []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L"}
	for i, id := range ids {
		if byID[id].Shape != wantShapes[i] {
			t.Errorf("%s: shape %v want %v", id, byID[id].Shape, wantShapes[i])
		}
	}
}

func TestParseClassDefAndAssignment(t *testing.T) {
	d, err := Parse("flowchart TB\nclassDef warn fill:#fee,stroke:#c00\nA[X]:::warn\nB[Y]\nclass B warn\n")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(d.ClassDefs) != 1 || d.ClassDefs[0].Name != "warn" {
		t.Fatalf("classDef: got %+v", d.ClassDefs)
	}
	if d.ClassDefs[0].Properties["fill"] != "#fee" || d.ClassDefs[0].Properties["stroke"] != "#c00" {
		t.Errorf("classDef properties: %+v", d.ClassDefs[0].Properties)
	}
	a := findNode(d, "A")
	b := findNode(d, "B")
	if !contains(a.ClassNames, "warn") {
		t.Errorf("A should have warn class: %+v", a)
	}
	if !contains(b.ClassNames, "warn") {
		t.Errorf("B should have warn class: %+v", b)
	}
}

func findNode(d *Diagram, id string) Node {
	for _, n := range d.Nodes {
		if n.ID == id {
			return n
		}
	}
	return Node{}
}
func contains(ss []string, s string) bool {
	for _, x := range ss {
		if x == s {
			return true
		}
	}
	return false
}
```

- [ ] **Step 3.2: Run, expect FAIL.**
- [ ] **Step 3.3: Extend `flowchart/parser.go`** with shape patterns, edge regex, classDef/class/style statements:

```go
// Append to parser.go:

// Node-shape patterns. Order matters: multi-char delimiters first so
// `A((text))` doesn't match the `A(text)` pattern.
var shapePatterns = []struct {
	re    *regexp.Regexp
	shape NodeShape
}{
	{regexp.MustCompile(`^([\w-]+)\(\(\((.+?)\)\)\)`), ShapeDoubleCircle},   // A(((text)))
	{regexp.MustCompile(`^([\w-]+)\(\[(.+?)\]\)`), ShapeStadium},            // A([text])
	{regexp.MustCompile(`^([\w-]+)\(\((.+?)\)\)`), ShapeCircle},             // A((text))
	{regexp.MustCompile(`^([\w-]+)\[\[(.+?)\]\]`), ShapeSubroutine},         // A[[text]]
	{regexp.MustCompile(`^([\w-]+)\[\((.+?)\)\]`), ShapeCylinder},           // A[(text)]
	{regexp.MustCompile(`^([\w-]+)\[/(.+?)\\\]`), ShapeTrapezoid},           // A[/text\]
	{regexp.MustCompile(`^([\w-]+)\[\\(.+?)/\]`), ShapeTrapezoidAlt},        // A[\text/]
	{regexp.MustCompile(`^([\w-]+)>(.+?)\]`), ShapeAsymmetric},              // A>text]
	{regexp.MustCompile(`^([\w-]+)\{\{(.+?)\}\}`), ShapeHexagon},            // A{{text}}
	{regexp.MustCompile(`^([\w-]+)\[(.+?)\]`), ShapeRect},                   // A[text]
	{regexp.MustCompile(`^([\w-]+)\((.+?)\)`), ShapeRound},                  // A(text)
	{regexp.MustCompile(`^([\w-]+)\{(.+?)\}`), ShapeDiamond},                // A{text}
}

// Class shorthand suffix `:::name`.
var classShorthandRE = regexp.MustCompile(`^:::([A-Za-z_][\w-]*)`)

// Bare node reference.
var bareRE = regexp.MustCompile(`^([\w-]+)`)

// Edge regex. Matches: <-?->|<-?-, <?=?=>|<?=?=, <?-\.+->|<?-\.+-,
// with optional `-- label --` or `|label|` segments.
//
// Implementation: try each variant in order.
var (
	edgeOps = []struct {
		op    string
		style EdgeStyle
		end   bool
		bidir bool
	}{
		{"<-->", EdgeSolid, true, true},
		{"-->", EdgeSolid, true, false},
		{"---", EdgeSolid, false, false},
		{"<==>", EdgeThick, true, true},
		{"==>", EdgeThick, true, false},
		{"===", EdgeThick, false, false},
		{"<-.->", EdgeDotted, true, true},
		{"-.->", EdgeDotted, true, false},
		{"-.-", EdgeDotted, false, false},
	}
)

// classDef name prop:val,prop:val
var classDefRE = regexp.MustCompile(`^classDef\s+([A-Za-z_][\w-]*)\s+(.+)$`)
var classAssignRE = regexp.MustCompile(`^class\s+([A-Za-z_][\w-]*(?:\s*,\s*[A-Za-z_][\w-]*)*)\s+([A-Za-z_][\w-]*)$`)
var styleAssignRE = regexp.MustCompile(`^style\s+([A-Za-z_][\w-]*)\s+(.+)$`)
var subgraphStartRE = regexp.MustCompile(`^subgraph\s+(.+)$`)
var subgraphEndRE = regexp.MustCompile(`^end$`)

// Replace the existing Parse stub body with a real walker that uses
// the helpers above. The body parser walks lines[1:], tracking the
// current subgraph stack, and routes each line:
//
//   - classDef → append to d.ClassDefs
//   - class A foo → set ClassNames for those nodes
//   - style A k:v,... → set d.NodeStyles[A]
//   - subgraph name [(label)?] → push a new subgraph onto the stack
//   - end → pop subgraph
//   - otherwise: try to parse as an edge statement; if no edge
//     operator, try as a single bare node declaration.
//
// The full parseBody function is ~150 LOC; below is the entry point.

func parseBody(d *Diagram, lines []string) error {
	stack := []*Subgraph{nil} // nil = top level
	nodes := map[string]int{}  // node ID → index into d.Nodes (so we update in place)

	addNode := func(n Node) {
		if existing, ok := nodes[n.ID]; ok {
			// Merge: prefer non-empty Label/Shape; append ClassNames.
			cur := &d.Nodes[existing]
			if n.Label != "" {
				cur.Label = n.Label
			}
			if n.Shape != cur.Shape && n.Shape != ShapeRect {
				cur.Shape = n.Shape
			}
			cur.ClassNames = append(cur.ClassNames, n.ClassNames...)
			return
		}
		nodes[n.ID] = len(d.Nodes)
		d.Nodes = append(d.Nodes, n)
		if cur := stack[len(stack)-1]; cur != nil {
			cur.NodeIDs = append(cur.NodeIDs, n.ID)
		}
	}

	for li, line := range lines {
		switch {
		case classDefRE.MatchString(line):
			m := classDefRE.FindStringSubmatch(line)
			d.ClassDefs = append(d.ClassDefs, ClassDef{
				Name:       m[1],
				Properties: parseProps(m[2]),
			})
		case classAssignRE.MatchString(line):
			m := classAssignRE.FindStringSubmatch(line)
			ids := splitCommaList(m[1])
			for _, id := range ids {
				if idx, ok := nodes[id]; ok {
					d.Nodes[idx].ClassNames = append(d.Nodes[idx].ClassNames, m[2])
				} else {
					// Forward reference: create a default rect node and assign.
					addNode(Node{ID: id, Label: id, Shape: ShapeRect, ClassNames: []string{m[2]}})
				}
			}
		case styleAssignRE.MatchString(line):
			m := styleAssignRE.FindStringSubmatch(line)
			d.NodeStyles[m[1]] = parseProps(m[2])
		case subgraphStartRE.MatchString(line):
			m := subgraphStartRE.FindStringSubmatch(line)
			id, label := parseSubgraphHeader(m[1])
			sg := &Subgraph{ID: id, Label: label}
			if cur := stack[len(stack)-1]; cur != nil {
				cur.Children = append(cur.Children, sg)
			} else {
				d.Subgraphs = append(d.Subgraphs, sg)
			}
			stack = append(stack, sg)
		case subgraphEndRE.MatchString(line):
			if len(stack) <= 1 {
				return fmt.Errorf("flowchart: line %d: unexpected `end`", li+2)
			}
			stack = stack[:len(stack)-1]
		default:
			if err := parseEdgeOrNode(line, addNode, func(e Edge) {
				d.Edges = append(d.Edges, e)
			}); err != nil {
				return fmt.Errorf("flowchart: line %d: %w", li+2, err)
			}
		}
	}
	if len(stack) > 1 {
		return fmt.Errorf("flowchart: %d unclosed subgraph(s)", len(stack)-1)
	}
	return nil
}

// parseEdgeOrNode tries to parse an edge first (because a node decl
// is a strict prefix of an edge decl). Falls back to a bare node.
func parseEdgeOrNode(line string, addNode func(Node), addEdge func(Edge)) error {
	for _, op := range edgeOps {
		if idx := strings.Index(line, op.op); idx >= 0 {
			leftRaw := strings.TrimSpace(line[:idx])
			rightRaw := strings.TrimSpace(line[idx+len(op.op):])
			leftNode, leftLabel, err := parseNodeDecl(leftRaw)
			if err != nil {
				return err
			}
			// Pipe-delimited edge label after operator: `--> |label|`
			label := leftLabel
			if strings.HasPrefix(rightRaw, "|") {
				if end := strings.Index(rightRaw[1:], "|"); end >= 0 {
					label = rightRaw[1 : 1+end]
					rightRaw = strings.TrimSpace(rightRaw[1+end+1:])
				}
			}
			// `--text-->` form: leftLabel parsed if `parseNodeDecl`
			// found a trailing `-- text` segment. (Implementation:
			// before calling parseNodeDecl, strip `-- ... --` from
			// just-before the operator and treat it as the label.)
			rightNode, _, err := parseNodeDecl(rightRaw)
			if err != nil {
				return err
			}
			addNode(leftNode)
			addNode(rightNode)
			addEdge(Edge{
				From:       leftNode.ID,
				To:         rightNode.ID,
				Label:      label,
				Style:      op.style,
				ArrowStart: op.bidir,
				ArrowEnd:   op.end,
			})
			return nil
		}
	}
	// Bare node declaration.
	n, _, err := parseNodeDecl(line)
	if err != nil {
		return err
	}
	addNode(n)
	return nil
}

// parseNodeDecl parses a single node spec like `A[label]:::warn` or
// `A`. Returns the Node and any extracted edge-label (for
// `A -- text -->` shorthand the caller pre-strips that segment).
func parseNodeDecl(s string) (Node, string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return Node{}, "", fmt.Errorf("empty node declaration")
	}

	// Try shape patterns.
	for _, sp := range shapePatterns {
		if m := sp.re.FindStringSubmatch(s); m != nil {
			n := Node{ID: m[1], Label: m[2], Shape: sp.shape}
			rest := s[len(m[0]):]
			if cm := classShorthandRE.FindStringSubmatch(rest); cm != nil {
				n.ClassNames = append(n.ClassNames, cm[1])
			}
			return n, "", nil
		}
	}
	// Bare ID.
	if m := bareRE.FindStringSubmatch(s); m != nil {
		n := Node{ID: m[1], Label: m[1], Shape: ShapeRect}
		rest := s[len(m[0]):]
		if cm := classShorthandRE.FindStringSubmatch(rest); cm != nil {
			n.ClassNames = append(n.ClassNames, cm[1])
		}
		return n, "", nil
	}
	return Node{}, "", fmt.Errorf("not a node declaration: %q", s)
}

// parseProps parses `key:value,key:value` into a map. Whitespace
// around keys/values is trimmed.
func parseProps(s string) map[string]string {
	m := map[string]string{}
	for _, kv := range strings.Split(s, ",") {
		parts := strings.SplitN(kv, ":", 2)
		if len(parts) != 2 {
			continue
		}
		m[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
	}
	return m
}

func splitCommaList(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		out = append(out, strings.TrimSpace(p))
	}
	return out
}

func parseSubgraphHeader(rest string) (id, label string) {
	rest = strings.TrimSpace(rest)
	// Forms: `subgraph X` or `subgraph X[Label]` or `subgraph X "Label"`.
	if i := strings.IndexAny(rest, "[\""); i >= 0 {
		return strings.TrimSpace(rest[:i]), strings.Trim(rest[i:], "[]\"")
	}
	return rest, rest
}
```

Wire `parseBody(d, lines[1:])` into the existing `Parse` function (replace the `_ = lines[1:]`).

- [ ] **Step 3.4: Run, expect PASS** for all parser tests.
- [ ] **Step 3.5: Commit** — `git commit -m "flowchart: parse nodes, edges, classDef, subgraphs"`

---

### Task 4: Parser — `-- label -->` shorthand

The `parseEdgeOrNode` above doesn't yet handle the `A -- text --> B` form (label between the two halves of the operator).

**Files:** Modify `flowchart/parser.go` and `parser_test.go`.

- [ ] **Step 4.1: Write the failing test**

```go
func TestParseEdgeLabelShortDashes(t *testing.T) {
	d, _ := Parse("flowchart TB\nA -- yes --> B\nA == thick === B\n")
	if d.Edges[0].Label != "yes" {
		t.Errorf("dash form: label %q", d.Edges[0].Label)
	}
	if d.Edges[0].ArrowEnd != true {
		t.Errorf("dash form: should have end arrow")
	}
	if d.Edges[1].Label != "thick" {
		t.Errorf("equals form: label %q", d.Edges[1].Label)
	}
	if d.Edges[1].Style != EdgeThick {
		t.Errorf("equals form: style %v", d.Edges[1].Style)
	}
}
```

- [ ] **Step 4.2: Run, expect FAIL.**
- [ ] **Step 4.3: Add a pre-pass in `parseEdgeOrNode`** that detects `-- text -->` / `== text ==>` / `-. text .->` segments and rewrites them to the `|label|` form before the operator-search loop. Implementation sketch:

```go
// Inside parseEdgeOrNode, before the operator loop:
line = collapseEdgeLabel(line)

// collapseEdgeLabel rewrites:
//   "A -- yes --> B"   → "A --> |yes| B"
//   "A == thick ==> B" → "A ==> |thick| B"
//   "A -. dotty .-> B" → "A -.-> |dotty| B"
// so the operator loop above finds the edge token cleanly.
func collapseEdgeLabel(line string) string {
	patterns := []struct {
		re *regexp.Regexp
		op string
	}{
		{regexp.MustCompile(`(<?-+)\s+(.+?)\s+(-+>?)`), ""},   // dash variants
		{regexp.MustCompile(`(<?={2,})\s+(.+?)\s+(={2,}>?)`), ""}, // equals variants
		{regexp.MustCompile(`(<?-?\.+)\s+(.+?)\s+(\.+-?>?)`), ""}, // dotted variants
	}
	for _, p := range patterns {
		line = p.re.ReplaceAllStringFunc(line, func(match string) string {
			m := p.re.FindStringSubmatch(match)
			return m[1] + m[3] + " |" + m[2] + "|"
		})
	}
	return line
}
```

This is intentionally lossy (pre-collapsing simplifies the operator search). Verify the test passes; if a subsequent diagram reveals a mis-rewrite, refine.

- [ ] **Step 4.4: Run, expect PASS.**
- [ ] **Step 4.5: Commit** — `git commit -m "flowchart: edge label shorthand --label-->"`

---

### Task 5: Layout — node sizing

The layout stage measures each node's label, computes the bbox (text + padding), and feeds nodes/edges to autog.

**Files:** Create `flowchart/shapes.go`, `flowchart/layout.go`, `flowchart/layout_test.go`.

- [ ] **Step 5.1: Write the failing test**

```go
// flowchart/layout_test.go
package flowchart

import (
	"testing"

	"github.com/luo-studio/go-mermaid/displaylist"
	"github.com/luo-studio/go-mermaid"
)

func TestLayoutSimpleChain(t *testing.T) {
	d, _ := Parse("flowchart TB\nA --> B\nB --> C\n")
	dl := Layout(d, mermaid.LayoutOptions{Measurer: fixedMeasurer{}})
	if dl == nil {
		t.Fatal("Layout returned nil")
	}
	// Three nodes → at least 3 Shape items + 2 Edge items.
	shapeCount, edgeCount := 0, 0
	for _, it := range dl.Items {
		switch it.(type) {
		case displaylist.Shape:
			shapeCount++
		case displaylist.Edge:
			edgeCount++
		}
	}
	if shapeCount != 3 {
		t.Errorf("shapes: got %d want 3", shapeCount)
	}
	if edgeCount != 2 {
		t.Errorf("edges: got %d want 2", edgeCount)
	}
	if dl.Width <= 0 || dl.Height <= 0 {
		t.Errorf("dl bbox: %vx%v", dl.Width, dl.Height)
	}
}

// fixedMeasurer reports 7px per char, 14px line height — deterministic
// for golden tests.
type fixedMeasurer struct{}

func (fixedMeasurer) Measure(text string, role displaylist.Role) (float64, float64) {
	return float64(len(text)) * 7, 14
}
```

- [ ] **Step 5.2: Run, expect FAIL** (Layout undefined).
- [ ] **Step 5.3: Implement `flowchart/shapes.go`**

```go
package flowchart

import "github.com/luo-studio/go-mermaid/displaylist"

// shapeKind maps the AST NodeShape to a DisplayList ShapeKind. Some
// shapes have no direct primitive; those return ShapeKindCustom and
// the caller must populate Path.
func shapeKind(s NodeShape) displaylist.ShapeKind {
	switch s {
	case ShapeRect, ShapeSubroutine:
		return displaylist.ShapeKindRect
	case ShapeRound:
		return displaylist.ShapeKindRound
	case ShapeStadium:
		return displaylist.ShapeKindStadium
	case ShapeDiamond:
		return displaylist.ShapeKindDiamond
	case ShapeCircle:
		return displaylist.ShapeKindCircle
	case ShapeDoubleCircle:
		return displaylist.ShapeKindDoubleCircle
	case ShapeHexagon:
		return displaylist.ShapeKindHexagon
	case ShapeCylinder:
		return displaylist.ShapeKindCylinder
	default:
		return displaylist.ShapeKindCustom
	}
}

// nodeSize returns the bbox (W, H) for a node given its label
// dimensions and shape. Padding varies by shape: rect/round use 16px;
// diamond/circle use 24px (rounder shapes need more breathing room).
func nodeSize(shape NodeShape, labelW, labelH float64) (w, h float64) {
	pad := 16.0
	switch shape {
	case ShapeDiamond, ShapeCircle, ShapeDoubleCircle, ShapeHexagon:
		pad = 24
	}
	w = labelW + pad*2
	h = labelH + pad
	// Circle shapes: square them up.
	if shape == ShapeCircle || shape == ShapeDoubleCircle {
		side := w
		if h > side {
			side = h
		}
		return side, side
	}
	return w, h
}
```

- [ ] **Step 5.4: Implement `flowchart/layout.go` (initial — no clusters, no custom shape paths)**

```go
package flowchart

import (
	"github.com/luo-studio/go-mermaid"
	"github.com/luo-studio/go-mermaid/autog"
	"github.com/luo-studio/go-mermaid/displaylist"
)

// Layout positions the diagram and emits a DisplayList. opts.Measurer
// (resolved through opts.measurer() via the mermaid package) is used
// to size each node's label.
func Layout(d *Diagram, opts mermaid.LayoutOptions) *displaylist.DisplayList {
	if d == nil {
		return &displaylist.DisplayList{}
	}
	measurer := opts.Measurer
	if measurer == nil {
		// Phase 2 callers always go through mermaid.ParseAndLayout
		// which fills in the default; this branch is defensive.
		measurer = (mermaid.LayoutOptions{}).Measurer // intentionally nil; layout falls through to a 7-per-char fallback below
	}
	measure := func(text string, role displaylist.Role) (float64, float64) {
		if measurer == nil {
			return float64(len(text)) * 7, 14
		}
		return measurer.Measure(text, role)
	}

	// Build autog input.
	var autogNodes []autog.Node
	for _, n := range d.Nodes {
		lw, lh := measure(n.Label, displaylist.RoleNode)
		w, h := nodeSize(n.Shape, lw, lh)
		autogNodes = append(autogNodes, autog.Node{ID: n.ID, Width: w, Height: h})
	}
	var autogEdges []autog.Edge
	for _, e := range d.Edges {
		autogEdges = append(autogEdges, autog.Edge{FromID: e.From, ToID: e.To})
	}

	out, err := autog.Layout(autog.Input{
		Nodes:        autogNodes,
		Edges:        autogEdges,
		Direction:    autogDir(d.Direction),
		NodeSpacing:  opts.NodeSpacing,
		LayerSpacing: opts.LayerSpacing,
	})
	if err != nil {
		// Layout failed; return an empty DisplayList so callers can
		// surface a fallback rather than panic.
		return &displaylist.DisplayList{}
	}

	// Build DisplayList.
	dl := &displaylist.DisplayList{Width: out.Width, Height: out.Height}
	posByID := map[string]autog.Node{}
	for _, n := range out.Nodes {
		posByID[n.ID] = n
	}
	astByID := map[string]Node{}
	for _, n := range d.Nodes {
		astByID[n.ID] = n
	}

	// Emit shapes + their labels.
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

	// Emit edges.
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
			mid := midpoint(points)
			dl.Items = append(dl.Items, displaylist.Text{
				Pos:    mid,
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

// customPath returns a polygon path for shapes without a native
// DisplayList kind.
func customPath(shape NodeShape, b displaylist.Rect) []displaylist.Point {
	switch shape {
	case ShapeAsymmetric:
		// Flag/banner: pentagon. Tip on the left.
		return []displaylist.Point{
			{X: b.X, Y: b.Y + b.H/2},
			{X: b.X + b.W*0.15, Y: b.Y},
			{X: b.X + b.W, Y: b.Y},
			{X: b.X + b.W, Y: b.Y + b.H},
			{X: b.X + b.W*0.15, Y: b.Y + b.H},
		}
	case ShapeTrapezoid:
		// `[/text\]` — wider top.
		off := b.W * 0.12
		return []displaylist.Point{
			{X: b.X + off, Y: b.Y},
			{X: b.X + b.W - off, Y: b.Y},
			{X: b.X + b.W, Y: b.Y + b.H},
			{X: b.X, Y: b.Y + b.H},
		}
	case ShapeTrapezoidAlt:
		off := b.W * 0.12
		return []displaylist.Point{
			{X: b.X, Y: b.Y},
			{X: b.X + b.W, Y: b.Y},
			{X: b.X + b.W - off, Y: b.Y + b.H},
			{X: b.X + off, Y: b.Y + b.H},
		}
	case ShapeParallelogram:
		off := b.W * 0.18
		return []displaylist.Point{
			{X: b.X + off, Y: b.Y},
			{X: b.X + b.W, Y: b.Y},
			{X: b.X + b.W - off, Y: b.Y + b.H},
			{X: b.X, Y: b.Y + b.H},
		}
	case ShapeParallelogramAlt:
		off := b.W * 0.18
		return []displaylist.Point{
			{X: b.X, Y: b.Y},
			{X: b.X + b.W - off, Y: b.Y},
			{X: b.X + b.W, Y: b.Y + b.H},
			{X: b.X + off, Y: b.Y + b.H},
		}
	}
	// Fallback: rectangle.
	return []displaylist.Point{
		{X: b.X, Y: b.Y},
		{X: b.X + b.W, Y: b.Y},
		{X: b.X + b.W, Y: b.Y + b.H},
		{X: b.X, Y: b.Y + b.H},
	}
}
```

- [ ] **Step 5.5: Run, expect PASS.**
- [ ] **Step 5.6: Commit** — `git commit -m "flowchart: layout via autog + DisplayList emit"`

---

### Task 6: Wire flowchart into top-level mermaid.ParseAndLayout

**Files:** Modify `mermaid.go`, `mermaid_test.go`.

- [ ] **Step 6.1: Update the test.** In `mermaid_test.go`, change `TestParseAndLayoutKnownButUnimplemented` to test sequence (still unimplemented) instead of flowchart, and add a positive test for flowchart:

```go
func TestParseAndLayoutFlowchart(t *testing.T) {
	dl, err := ParseAndLayout("flowchart TB\nA --> B\n", LayoutOptions{})
	if err != nil {
		t.Fatalf("ParseAndLayout: %v", err)
	}
	if dl == nil || len(dl.Items) == 0 {
		t.Fatal("expected non-empty DisplayList")
	}
}

func TestParseAndLayoutSequenceUnimplemented(t *testing.T) {
	_, err := ParseAndLayout("sequenceDiagram\nA->>B: hi\n", LayoutOptions{})
	if !errors.Is(err, ErrNotImplemented) {
		t.Fatalf("expected ErrNotImplemented, got %v", err)
	}
}
```

- [ ] **Step 6.2: Update `mermaid.go`'s dispatcher**:

```go
import "github.com/luo-studio/go-mermaid/flowchart"

// ...

func ParseAndLayout(src string, opts LayoutOptions) (*displaylist.DisplayList, error) {
	switch detectType(src) {
	case typeFlowchart:
		ast, err := flowchart.Parse(src)
		if err != nil {
			return nil, err
		}
		// Ensure default measurer if caller didn't supply one.
		if opts.Measurer == nil {
			opts.Measurer = opts.measurer()
		}
		return flowchart.Layout(ast, opts), nil
	case typeSequence, typeClass, typeER, typeState:
		return nil, ErrNotImplemented
	default:
		return nil, ErrUnknownDiagram
	}
}
```

- [ ] **Step 6.3: Run, expect PASS.**
- [ ] **Step 6.4: Commit** — `git commit -m "mermaid: wire flowchart into ParseAndLayout"`

---

### Task 7: PDF emitter — Style + DrawInto skeleton

**Files:** Create `pdf/emit.go`, `pdf/style.go`, `pdf/emit_test.go`.

- [ ] **Step 7.1: Add fpdf dep**

```bash
go get codeberg.org/go-pdf/fpdf
```

(Use the platform's `replace` if needed: `replace codeberg.org/go-pdf/fpdf => github.com/luo-studio/fpdf <pinned-version>`. Match what `go-tex/go.mod` uses.)

- [ ] **Step 7.2: Write the failing test**

```go
// pdf/emit_test.go
package mermaidpdf

import (
	"bytes"
	"testing"

	"codeberg.org/go-pdf/fpdf"
)

func TestDrawMermaidSimpleFlowchart(t *testing.T) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Helvetica", "", 10)

	src := "flowchart TB\nA --> B\nB --> C\n"
	if err := DrawMermaid(pdf, src, 10, 30, EmbedDefaults()); err != nil {
		t.Fatalf("DrawMermaid: %v", err)
	}
	if pdf.Err() {
		t.Fatalf("fpdf error: %v", pdf.Error())
	}
	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		t.Fatalf("Output: %v", err)
	}
	if !bytes.HasPrefix(buf.Bytes(), []byte("%PDF-")) {
		t.Fatal("output is not a PDF")
	}
	if buf.Len() < 1000 {
		t.Fatalf("PDF unexpectedly small: %d bytes", buf.Len())
	}
}

func TestDrawMermaidUnknownDiagram(t *testing.T) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	if err := DrawMermaid(pdf, "garbage\n", 10, 10, EmbedDefaults()); err == nil {
		t.Fatal("expected error for non-diagram input")
	}
}
```

- [ ] **Step 7.3: Run, expect FAIL** (package doesn't exist).
- [ ] **Step 7.4: Implement `pdf/style.go`**

```go
// Package mermaidpdf renders go-mermaid DisplayLists into fpdf
// documents. The package is imported as `mermaidpdf` to avoid
// stdlib `pdf` collisions.
package mermaidpdf

import "github.com/luo-studio/go-mermaid/displaylist"

// RoleStyle is the visual appearance of a single semantic Role.
type RoleStyle struct {
	StrokeR, StrokeG, StrokeB float64 // 0-255; -1 = no stroke
	StrokeWidth                float64 // mm; ignored if no stroke
	DashPattern                []float64

	FillR, FillG, FillB float64 // 0-255; -1 = no fill (transparent)

	TextR, TextG, TextB float64
	Font                string  // fpdf font family
	FontStyle           string  // "", "B", "I", "BI"
	FontSize            float64 // points
}

// Style is the per-Role visual map. Default is used as fallback.
type Style struct {
	Roles   map[displaylist.Role]RoleStyle
	Default RoleStyle
}

// lookup returns the style for role, falling back to Default.
func (s Style) lookup(role displaylist.Role) RoleStyle {
	if rs, ok := s.Roles[role]; ok {
		return rs
	}
	return s.Default
}

// DefaultStyle returns a sensible black-on-white style suitable for
// simple flowcharts.
func DefaultStyle() Style {
	body := RoleStyle{
		StrokeR: 30, StrokeG: 30, StrokeB: 30,
		StrokeWidth: 0.3,
		FillR:       -1, FillG: -1, FillB: -1, // transparent
		TextR: 0, TextG: 0, TextB: 0,
		Font: "Helvetica", FontStyle: "", FontSize: 10,
	}
	bodyBold := body
	bodyBold.FontStyle = "B"
	muted := body
	muted.TextR, muted.TextG, muted.TextB = 100, 100, 100

	return Style{
		Default: body,
		Roles: map[displaylist.Role]RoleStyle{
			displaylist.RoleNode:         body,
			displaylist.RoleEdge:         body,
			displaylist.RoleEdgeLabel:    muted,
			displaylist.RoleSubgraph:     body,
			displaylist.RoleClusterTitle: bodyBold,
		},
	}
}
```

- [ ] **Step 7.5: Implement `pdf/emit.go` (skeleton)**

```go
package mermaidpdf

import (
	"fmt"

	"codeberg.org/go-pdf/fpdf"

	mermaid "github.com/luo-studio/go-mermaid"
	"github.com/luo-studio/go-mermaid/displaylist"
)

// EmbedOptions configures DrawMermaid / DrawInto.
type EmbedOptions struct {
	// Style maps DisplayList roles to fpdf colors/fonts/widths.
	Style Style

	// Layout options forwarded to mermaid.ParseAndLayout.
	Layout mermaid.LayoutOptions

	// MaxWidth caps rendered width in mm (the unit fpdf is using).
	// If laid-out DisplayList exceeds it, uniform-scale down. 0 = no cap.
	MaxWidth float64

	// Padding around the diagram in mm. Default 0.
	Padding float64
}

// EmbedDefaults returns sensible defaults: DefaultStyle, no MaxWidth.
func EmbedDefaults() EmbedOptions {
	return EmbedOptions{Style: DefaultStyle()}
}

// DrawMermaid is the one-call helper: parse → layout → draw at (x, y).
func DrawMermaid(pdf *fpdf.Fpdf, src string, x, y float64, opts EmbedOptions) error {
	dl, err := mermaid.ParseAndLayout(src, opts.Layout)
	if err != nil {
		return err
	}
	return DrawInto(pdf, dl, x, y, opts)
}

// DrawInto draws an already-laid-out DisplayList into pdf at (x, y).
// Coordinates inside dl are translated by (x, y) and optionally
// scaled to MaxWidth.
func DrawInto(pdf *fpdf.Fpdf, dl *displaylist.DisplayList, x, y float64, opts EmbedOptions) error {
	if dl == nil || len(dl.Items) == 0 {
		return nil
	}
	style := opts.Style
	if style.Default == (RoleStyle{}) && len(style.Roles) == 0 {
		style = DefaultStyle()
	}

	// Pixel-to-mm scale: the layout produces sizes in DisplayList
	// units which we treat as 1 unit = 1 mm. Some callers will want
	// to tighten this; expose later via opts.UnitsPerMm if needed.
	const unitsPerMm = 1.0
	scale := 1.0 / unitsPerMm
	if opts.MaxWidth > 0 && dl.Width*scale > opts.MaxWidth {
		scale = opts.MaxWidth / dl.Width
	}

	dx := x + opts.Padding
	dy := y + opts.Padding

	tx := func(p displaylist.Point) (float64, float64) { return dx + p.X*scale, dy + p.Y*scale }
	tr := func(r displaylist.Rect) (float64, float64, float64, float64) {
		return dx + r.X*scale, dy + r.Y*scale, r.W * scale, r.H * scale
	}

	for _, it := range dl.Items {
		switch v := it.(type) {
		case displaylist.Shape:
			drawShape(pdf, v, tr, style.lookup(v.Role))
		case displaylist.Edge:
			drawEdge(pdf, v, tx, style.lookup(v.Role), scale)
		case displaylist.Text:
			drawText(pdf, v, tx, style.lookup(v.Role))
		case displaylist.Cluster:
			// Phase 3 wires this; for Phase 2 ignore.
		case displaylist.Marker:
			// Inline arrow markers handled in Edge; standalone
			// markers are rare. Ignore for Phase 2.
		default:
			return fmt.Errorf("mermaidpdf: unknown DisplayList item kind %T", v)
		}
	}
	return nil
}
```

- [ ] **Step 7.6: Run.** Test will still fail (drawShape/drawEdge/drawText undefined). Move on to next task.
- [ ] **Step 7.7: Commit** — `git commit -m "mermaidpdf: DrawMermaid/DrawInto/Style skeleton"`

---

### Task 8: PDF emitter — drawShape

**Files:** Create `pdf/draw_shape.go`.

- [ ] **Step 8.1: Implement**

```go
package mermaidpdf

import (
	"codeberg.org/go-pdf/fpdf"

	"github.com/luo-studio/go-mermaid/displaylist"
)

// drawShape renders a Shape primitive into the pdf. tr translates a
// DisplayList Rect to pdf-space (x, y, w, h).
func drawShape(pdf *fpdf.Fpdf, s displaylist.Shape, tr func(displaylist.Rect) (float64, float64, float64, float64), rs RoleStyle) {
	x, y, w, h := tr(s.BBox)
	applyStroke(pdf, rs)
	fillStyle := applyFill(pdf, rs)

	switch s.Kind {
	case displaylist.ShapeKindRect:
		pdf.Rect(x, y, w, h, fillStyle)
	case displaylist.ShapeKindRound:
		r := minf(8, w/2, h/2)
		pdf.RoundedRect(x, y, w, h, r, "1234", fillStyle)
	case displaylist.ShapeKindStadium:
		r := h / 2
		pdf.RoundedRect(x, y, w, h, r, "1234", fillStyle)
	case displaylist.ShapeKindCircle, displaylist.ShapeKindDoubleCircle:
		cx, cy := x+w/2, y+h/2
		r := w / 2
		pdf.Ellipse(cx, cy, r, r, 0, fillStyle)
		if s.Kind == displaylist.ShapeKindDoubleCircle {
			pdf.Ellipse(cx, cy, r-2, r-2, 0, fillStyle)
		}
	case displaylist.ShapeKindEllipse:
		pdf.Ellipse(x+w/2, y+h/2, w/2, h/2, 0, fillStyle)
	case displaylist.ShapeKindDiamond:
		pdf.Polygon([]fpdf.PointType{
			{X: x + w/2, Y: y},
			{X: x + w, Y: y + h/2},
			{X: x + w/2, Y: y + h},
			{X: x, Y: y + h/2},
		}, fillStyle)
	case displaylist.ShapeKindHexagon:
		off := w * 0.18
		pdf.Polygon([]fpdf.PointType{
			{X: x + off, Y: y},
			{X: x + w - off, Y: y},
			{X: x + w, Y: y + h/2},
			{X: x + w - off, Y: y + h},
			{X: x + off, Y: y + h},
			{X: x, Y: y + h/2},
		}, fillStyle)
	case displaylist.ShapeKindCylinder:
		// Top ellipse + side rect + bottom ellipse half.
		ry := h * 0.12
		pdf.Rect(x, y+ry, w, h-ry*2, fillStyle)
		pdf.Ellipse(x+w/2, y+ry, w/2, ry, 0, fillStyle)
		pdf.Ellipse(x+w/2, y+h-ry, w/2, ry, 0, fillStyle)
	case displaylist.ShapeKindCustom:
		if len(s.Path) < 3 {
			return
		}
		pts := make([]fpdf.PointType, len(s.Path))
		for i, p := range s.Path {
			px, py := translatePoint(p, tr, s.BBox)
			pts[i] = fpdf.PointType{X: px, Y: py}
		}
		pdf.Polygon(pts, fillStyle)
	}
}

// translatePoint maps a Path point (which is in DisplayList absolute
// coords) to pdf coords. Implementation: build a Rect with origin at
// the path point and zero W/H, run tr, take the upper-left.
func translatePoint(p displaylist.Point, tr func(displaylist.Rect) (float64, float64, float64, float64), _ displaylist.Rect) (float64, float64) {
	x, y, _, _ := tr(displaylist.Rect{X: p.X, Y: p.Y, W: 0, H: 0})
	return x, y
}

// applyStroke configures fpdf draw color / line width / dash from rs.
func applyStroke(pdf *fpdf.Fpdf, rs RoleStyle) {
	if rs.StrokeR < 0 {
		pdf.SetLineWidth(0)
		return
	}
	pdf.SetDrawColor(int(rs.StrokeR), int(rs.StrokeG), int(rs.StrokeB))
	w := rs.StrokeWidth
	if w <= 0 {
		w = 0.3
	}
	pdf.SetLineWidth(w)
	if len(rs.DashPattern) >= 2 {
		pdf.SetDashPattern(rs.DashPattern, 0)
	} else {
		pdf.SetDashPattern(nil, 0)
	}
}

// applyFill configures fpdf fill color from rs and returns the
// fill-style code: "F" (fill), "FD" (fill+draw), or "D" (draw only).
func applyFill(pdf *fpdf.Fpdf, rs RoleStyle) string {
	hasFill := rs.FillR >= 0
	hasStroke := rs.StrokeR >= 0
	if hasFill {
		pdf.SetFillColor(int(rs.FillR), int(rs.FillG), int(rs.FillB))
	}
	switch {
	case hasFill && hasStroke:
		return "FD"
	case hasFill:
		return "F"
	default:
		return "D"
	}
}

func minf(a, b, c float64) float64 {
	m := a
	if b < m {
		m = b
	}
	if c < m {
		m = c
	}
	return m
}
```

- [ ] **Step 8.2: Commit** — `git commit -m "mermaidpdf: draw_shape per ShapeKind"`

---

### Task 9: PDF emitter — drawEdge + drawText

**Files:** Create `pdf/draw_edge.go`, `pdf/draw_text.go`.

- [ ] **Step 9.1: Implement `pdf/draw_edge.go`**

```go
package mermaidpdf

import (
	"math"

	"codeberg.org/go-pdf/fpdf"

	"github.com/luo-studio/go-mermaid/displaylist"
)

func drawEdge(pdf *fpdf.Fpdf, e displaylist.Edge, tx func(displaylist.Point) (float64, float64), rs RoleStyle, scale float64) {
	if len(e.Points) < 2 {
		return
	}
	applyStroke(pdf, rs)
	if e.LineStyle == displaylist.LineStyleDashed || e.LineStyle == displaylist.LineStyleDotted {
		dash := []float64{1.5, 1.5}
		if e.LineStyle == displaylist.LineStyleDotted {
			dash = []float64{0.4, 1.0}
		}
		pdf.SetDashPattern(dash, 0)
		defer pdf.SetDashPattern(nil, 0)
	}
	if e.LineStyle == displaylist.LineStyleThick {
		pdf.SetLineWidth(maxf(rs.StrokeWidth*2.5, 0.6))
		defer pdf.SetLineWidth(rs.StrokeWidth)
	}

	// Polyline.
	x0, y0 := tx(e.Points[0])
	for _, p := range e.Points[1:] {
		x1, y1 := tx(p)
		pdf.Line(x0, y0, x1, y1)
		x0, y0 = x1, y1
	}

	// Arrow heads.
	if e.ArrowEnd != displaylist.MarkerNone {
		drawArrow(pdf, e.Points[len(e.Points)-1], e.Points[len(e.Points)-2], e.ArrowEnd, tx, rs, scale)
	}
	if e.ArrowStart != displaylist.MarkerNone {
		drawArrow(pdf, e.Points[0], e.Points[1], e.ArrowStart, tx, rs, scale)
	}
}

// drawArrow renders an arrow marker at `tip` pointing away from
// `behind`. kind selects the marker glyph.
func drawArrow(pdf *fpdf.Fpdf, tip, behind displaylist.Point, kind displaylist.MarkerKind, tx func(displaylist.Point) (float64, float64), rs RoleStyle, scale float64) {
	tx0, ty0 := tx(behind)
	tx1, ty1 := tx(tip)
	dx, dy := tx1-tx0, ty1-ty0
	d := math.Hypot(dx, dy)
	if d == 0 {
		return
	}
	ux, uy := dx/d, dy/d
	// Perpendicular.
	px, py := -uy, ux

	size := 2.5 * scale // mm
	switch kind {
	case displaylist.MarkerArrow, displaylist.MarkerArrowOpen:
		bx := tx1 - ux*size
		by := ty1 - uy*size
		l := fpdf.PointType{X: bx + px*size*0.5, Y: by + py*size*0.5}
		r := fpdf.PointType{X: bx - px*size*0.5, Y: by - py*size*0.5}
		pdf.Polygon([]fpdf.PointType{
			{X: tx1, Y: ty1}, l, r,
		}, ternary(kind == displaylist.MarkerArrow, "F", "D"))
	case displaylist.MarkerCross:
		// X mark.
		pdf.Line(tx1-px*size*0.5-ux*size*0.5, ty1-py*size*0.5-uy*size*0.5,
			tx1+px*size*0.5+ux*size*0.5, ty1+py*size*0.5+uy*size*0.5)
		pdf.Line(tx1+px*size*0.5-ux*size*0.5, ty1+py*size*0.5-uy*size*0.5,
			tx1-px*size*0.5+ux*size*0.5, ty1-py*size*0.5+uy*size*0.5)
	}
}

func ternary(b bool, a, b2 string) string {
	if b {
		return a
	}
	return b2
}

func maxf(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
```

- [ ] **Step 9.2: Implement `pdf/draw_text.go`**

```go
package mermaidpdf

import (
	"codeberg.org/go-pdf/fpdf"

	"github.com/luo-studio/go-mermaid/displaylist"
)

func drawText(pdf *fpdf.Fpdf, t displaylist.Text, tx func(displaylist.Point) (float64, float64), rs RoleStyle) {
	if len(t.Lines) == 0 {
		return
	}
	if rs.Font == "" {
		rs.Font = "Helvetica"
	}
	if rs.FontSize <= 0 {
		rs.FontSize = 10
	}
	pdf.SetFont(rs.Font, rs.FontStyle, rs.FontSize)
	pdf.SetTextColor(int(rs.TextR), int(rs.TextG), int(rs.TextB))

	x, y := tx(t.Pos)
	// fpdf's text origin is the baseline. Approx baseline = y +
	// fontHeightMM*0.4 when VAlign=middle and we're treating y as
	// vertical center.
	lineH := rs.FontSize * 0.4 // mm; 1pt ≈ 0.353 mm; FontSize is in pt
	totalH := lineH * float64(len(t.Lines))
	baseY := y - totalH/2 + lineH*0.7
	for _, line := range t.Lines {
		w := pdf.GetStringWidth(line)
		var lx float64
		switch t.Align {
		case displaylist.AlignLeft:
			lx = x
		case displaylist.AlignRight:
			lx = x - w
		default: // center
			lx = x - w/2
		}
		pdf.Text(lx, baseY, line)
		baseY += lineH
	}
}
```

- [ ] **Step 9.3: Run tests, expect PASS** for `TestDrawMermaidSimpleFlowchart` and `TestDrawMermaidUnknownDiagram`.
- [ ] **Step 9.4: Commit** — `git commit -m "mermaidpdf: drawEdge + drawText"`

---

### Task 10: Golden snapshot tests

**Files:** Create `testdata/flowchart/parse/*` and `testdata/flowchart/layout/*`. Add `flowchart/golden_test.go`.

- [ ] **Step 10.1: Add a small corpus.** For each fixture below, create both the `.mmd` source and the corresponding `.golden.json`. Generate the golden file by running the parser/layout once with `-update` flag.

Fixtures (≥3 each):
- `parse/simple-tb.mmd` — 3 nodes, 2 edges, TD direction.
- `parse/all-shapes.mmd` — one node per shape kind (12+).
- `parse/all-edges.mmd` — every operator variant.
- `parse/classdef.mmd` — classDef + class + ::: + style.
- `layout/simple-tb.mmd` — same as parse, but golden is the
  DisplayList JSON.
- `layout/all-shapes.mmd` — DisplayList with 12 shapes.

- [ ] **Step 10.2: Implement `flowchart/golden_test.go`**

```go
package flowchart

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/luo-studio/go-mermaid"
	"github.com/luo-studio/go-mermaid/displaylist"
)

var update = flag.Bool("update", false, "rewrite golden files")

func TestParseGolden(t *testing.T) {
	dir := "../testdata/flowchart/parse"
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".mmd") {
			continue
		}
		t.Run(e.Name(), func(t *testing.T) {
			src, _ := os.ReadFile(filepath.Join(dir, e.Name()))
			d, err := Parse(string(src))
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			out, _ := json.MarshalIndent(d, "", "  ")
			golden := strings.TrimSuffix(filepath.Join(dir, e.Name()), ".mmd") + ".golden.json"
			if *update {
				_ = os.WriteFile(golden, out, 0644)
				return
			}
			want, err := os.ReadFile(golden)
			if err != nil {
				t.Fatalf("read golden: %v", err)
			}
			if string(out) != strings.TrimRight(string(want), "\n") {
				t.Fatalf("ast mismatch — run with -update if intentional\nwant: %s\ngot:  %s", want, out)
			}
		})
	}
}

func TestLayoutGolden(t *testing.T) {
	dir := "../testdata/flowchart/layout"
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".mmd") {
			continue
		}
		t.Run(e.Name(), func(t *testing.T) {
			src, _ := os.ReadFile(filepath.Join(dir, e.Name()))
			d, err := Parse(string(src))
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			dl := Layout(d, mermaid.LayoutOptions{Measurer: testMeasurer{}})
			out, _ := json.MarshalIndent(dl, "", "  ")
			golden := strings.TrimSuffix(filepath.Join(dir, e.Name()), ".mmd") + ".golden.json"
			if *update {
				_ = os.WriteFile(golden, out, 0644)
				return
			}
			want, _ := os.ReadFile(golden)
			if string(out) != strings.TrimRight(string(want), "\n") {
				t.Fatalf("displaylist mismatch — run with -update if intentional")
			}
		})
	}
}

type testMeasurer struct{}

func (testMeasurer) Measure(text string, role displaylist.Role) (float64, float64) {
	return float64(len(text)) * 7, 14
}
```

- [ ] **Step 10.3: Run with -update once to seed:**
```bash
go test ./flowchart/... -run TestParseGolden -update
go test ./flowchart/... -run TestLayoutGolden -update
```
- [ ] **Step 10.4: Inspect each golden file** for sanity (no NaN, sane bbox, etc).
- [ ] **Step 10.5: Re-run without -update, expect PASS.**
- [ ] **Step 10.6: Commit** — `git add testdata/ flowchart/golden_test.go && git commit -m "flowchart: golden parse + layout snapshot tests"`

---

### Task 11: Property tests

**Files:** Create `flowchart/property_test.go`.

- [ ] **Step 11.1: Implement**

```go
package flowchart

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/luo-studio/go-mermaid"
	"github.com/luo-studio/go-mermaid/displaylist"
)

// TestPropertyNoPanic generates random valid flowcharts and
// asserts that Parse + Layout never panic and produce a
// well-formed DisplayList.
func TestPropertyNoPanic(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	for i := 0; i < 200; i++ {
		src := genFlowchart(rng)
		d, err := Parse(src)
		if err != nil {
			t.Fatalf("Parse(%q): %v", src, err)
		}
		dl := Layout(d, mermaid.LayoutOptions{Measurer: testMeasurer{}})
		if dl == nil {
			t.Fatalf("Layout returned nil for %q", src)
		}
		// Bbox containment: every Shape.BBox lies within [0, Width] × [0, Height].
		for _, it := range dl.Items {
			if s, ok := it.(displaylist.Shape); ok {
				if s.BBox.X < 0 || s.BBox.Y < 0 ||
					s.BBox.X+s.BBox.W > dl.Width+0.001 ||
					s.BBox.Y+s.BBox.H > dl.Height+0.001 {
					t.Fatalf("shape outside bbox: %+v in dl %vx%v", s, dl.Width, dl.Height)
				}
			}
		}
	}
}

// genFlowchart produces a small valid flowchart source. It biases
// toward simple chains and stars; intentionally avoids subgraphs
// (those land in Phase 3).
func genFlowchart(rng *rand.Rand) string {
	dirs := []string{"TB", "BT", "LR", "RL"}
	src := fmt.Sprintf("flowchart %s\n", dirs[rng.Intn(len(dirs))])
	n := 2 + rng.Intn(5)
	ids := make([]string, n)
	for i := range ids {
		ids[i] = fmt.Sprintf("N%d", i)
	}
	for i := 1; i < n; i++ {
		// chain
		op := []string{"-->", "---", "-.->", "==>"}[rng.Intn(4)]
		src += fmt.Sprintf("%s %s %s\n", ids[i-1], op, ids[i])
	}
	return src
}
```

- [ ] **Step 11.2: Run, expect PASS.**
- [ ] **Step 11.3: Commit** — `git commit -m "flowchart: property test (no panic + bbox containment)"`

---

### Task 12: Phase 2 smoke test + final vet

**Files:** Modify `mermaid_smoke_test.go`.

- [ ] **Step 12.1: Replace the smoke test:**

```go
package mermaid

import (
	"testing"

	"github.com/luo-studio/go-mermaid/displaylist"
)

func TestPhase2Smoke(t *testing.T) {
	src := "flowchart TB\nA[Start] --> B{Decide}\nB -- yes --> C[Do it]\nB -- no --> D[Skip]\n"
	dl, err := ParseAndLayout(src, LayoutOptions{})
	if err != nil {
		t.Fatalf("ParseAndLayout: %v", err)
	}
	shapes := 0
	for _, it := range dl.Items {
		if _, ok := it.(displaylist.Shape); ok {
			shapes++
		}
	}
	if shapes < 4 {
		t.Fatalf("expected ≥4 shapes, got %d", shapes)
	}
}
```

- [ ] **Step 12.2: `go test ./...` and `go vet ./...` — both clean.**
- [ ] **Step 12.3: Commit** — `git commit -m "mermaid: phase 2 smoke test"`

---

## Self-Review

| Spec section | Phase 2 task |
|---|---|
| Flowchart directions (TB/BT/LR/RL) + TD alias | Task 2 |
| Flowchart shapes (12+ kinds) | Tasks 1, 3, 5 |
| Edge styles (solid/dotted/thick) + bidirectional | Tasks 1, 3, 4 |
| Edge labels (`-- text -->`, `\|text\|`) | Task 4, Task 3 |
| classDef + class + ::: + style | Task 3 |
| Subgraph parsing (layout deferred) | Task 3 |
| autog-based layout | Task 5 |
| DisplayList emit (Shape/Edge/Text) | Task 5 |
| Top-level dispatch into flowchart | Task 6 |
| pdf.DrawMermaid one-call helper | Task 7 |
| pdf.DrawInto for inspect-then-draw | Task 7 |
| Per-shape native drawing | Task 8 |
| Edge polylines + arrowheads | Task 9 |
| Text rendering with role-driven font | Task 9 |
| Golden snapshot tests | Task 10 |
| Property tests (no panic + bbox) | Task 11 |
| Smoke test | Task 12 |

**Deferred to later phases:** Cluster (subgraph) layout (Phase 3); state-start/end pseudostates (Phase 7 inherits); ::: shorthand → role styling integration (Phase 5+ as roles are needed).

## Open Questions / Risks

- **fpdf font metrics vs. embedded Inter measurer**: callers using the default Measurer will get sizes that don't quite match what fpdf will draw with Helvetica. To get tight fit, callers should supply a Measurer wrapping `pdf.GetStringWidth`. Document this in `EmbedOptions` godoc.
- **Edge label collapsing regex** in Task 4 is heuristic. If real-world Mermaid samples reveal mis-rewrites, replace with a proper inline-aware tokenizer.
- **Custom-shape paths** for asymmetric/trapezoid/parallelogram are approximations; tweak the offsets in `customPath` after visual review.
