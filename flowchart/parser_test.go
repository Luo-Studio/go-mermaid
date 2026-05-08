package flowchart

import "testing"

func TestParseDirection(t *testing.T) {
	cases := map[string]Direction{
		"flowchart TB\n":             DirectionTB,
		"flowchart TD\n":             DirectionTB,
		"flowchart BT\n":             DirectionBT,
		"flowchart LR\n":             DirectionLR,
		"flowchart RL\n":             DirectionRL,
		"graph LR\n":                 DirectionLR,
		"  graph    LR  \n":          DirectionLR,
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

func TestParseEdgeLabelShortDashes(t *testing.T) {
	d, _ := Parse("flowchart TB\nA -- yes --> B\nA == thick ==> B\nA -. dotty .-> B\n")
	if d.Edges[0].Label != "yes" {
		t.Errorf("dash form: label %q", d.Edges[0].Label)
	}
	if !d.Edges[0].ArrowEnd {
		t.Errorf("dash form: should have end arrow")
	}
	if d.Edges[1].Label != "thick" {
		t.Errorf("equals form: label %q", d.Edges[1].Label)
	}
	if d.Edges[1].Style != EdgeThick {
		t.Errorf("equals form: style %v", d.Edges[1].Style)
	}
	if d.Edges[2].Label != "dotty" {
		t.Errorf("dotted form: label %q", d.Edges[2].Label)
	}
	if d.Edges[2].Style != EdgeDotted {
		t.Errorf("dotted form: style %v", d.Edges[2].Style)
	}
}

func TestParseSubgraph(t *testing.T) {
	src := `flowchart TB
subgraph one [Outer]
A --> B
end
A --> C
`
	d, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(d.Subgraphs) != 1 {
		t.Fatalf("want 1 subgraph, got %d", len(d.Subgraphs))
	}
	sg := d.Subgraphs[0]
	if sg.ID != "one" {
		t.Errorf("subgraph ID: %q", sg.ID)
	}
	if sg.Label != "Outer" {
		t.Errorf("subgraph label: %q", sg.Label)
	}
	if !contains(sg.NodeIDs, "A") || !contains(sg.NodeIDs, "B") {
		t.Errorf("subgraph members: %v", sg.NodeIDs)
	}
}

func TestParseStyleStatement(t *testing.T) {
	d, err := Parse("flowchart TB\nA[X]\nstyle A fill:#abc,stroke:#def\n")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	props := d.NodeStyles["A"]
	if props["fill"] != "#abc" || props["stroke"] != "#def" {
		t.Errorf("style props: %+v", props)
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
