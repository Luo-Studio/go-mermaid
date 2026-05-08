package class

import "testing"

func TestParseHeader(t *testing.T) {
	if _, err := Parse("classDiagram\n"); err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if _, err := Parse("notClass\n"); err == nil {
		t.Fatal("expected error for non-class header")
	}
}

func TestParseClassBlock(t *testing.T) {
	src := `classDiagram
class Animal {
  <<interface>>
  +name: string
  +eat() void
}
`
	d, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(d.Classes) != 1 {
		t.Fatalf("classes: %d", len(d.Classes))
	}
	c := d.Classes[0]
	if c.ID != "Animal" || c.Annotation != "<<interface>>" {
		t.Errorf("class: %+v", c)
	}
	if len(c.Members) != 2 {
		t.Fatalf("members: %d", len(c.Members))
	}
	if c.Members[0].IsMethod || c.Members[0].Name != "name" || c.Members[0].Visibility != VisPublic {
		t.Errorf("member 0: %+v", c.Members[0])
	}
	if !c.Members[1].IsMethod || c.Members[1].Name != "eat" {
		t.Errorf("member 1: %+v", c.Members[1])
	}
}

func TestParseRelationshipsAllVariants(t *testing.T) {
	src := `classDiagram
A <|-- B
C *-- D
E o-- F
G --> H
I ..> J
K ..|> L
M -- N
`
	d, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(d.Relationships) != 7 {
		t.Fatalf("relationships: %d", len(d.Relationships))
	}
	wantKinds := []RelationKind{
		RelInheritance, RelComposition, RelAggregation,
		RelAssociation, RelDependency, RelRealization, RelLink,
	}
	for i, want := range wantKinds {
		if d.Relationships[i].Kind != want {
			t.Errorf("rel %d: kind %v want %v", i, d.Relationships[i].Kind, want)
		}
	}
}

func TestParseCardinalityAndLabel(t *testing.T) {
	d, err := Parse(`classDiagram
A "1" --> "*" B : has
`)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	r := d.Relationships[0]
	if r.FromCard != "1" || r.ToCard != "*" {
		t.Errorf("cardinality: %+v", r)
	}
	if r.Label != "has" {
		t.Errorf("label: %q", r.Label)
	}
}

func TestParseInlineMember(t *testing.T) {
	d, _ := Parse("classDiagram\nclass Foo\nFoo : +bar() int\n")
	if len(d.Classes) != 1 {
		t.Fatalf("classes: %d", len(d.Classes))
	}
	if len(d.Classes[0].Members) != 1 || !d.Classes[0].Members[0].IsMethod {
		t.Errorf("members: %+v", d.Classes[0].Members)
	}
}

func TestParseNamespace(t *testing.T) {
	src := `classDiagram
namespace Net {
class HTTP
class TCP
}
HTTP --> TCP
`
	d, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(d.Namespaces) != 1 {
		t.Fatalf("namespaces: %d", len(d.Namespaces))
	}
	ns := d.Namespaces[0]
	if ns.Name != "Net" || len(ns.ClassIDs) != 2 {
		t.Errorf("namespace: %+v", ns)
	}
}
