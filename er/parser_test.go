package er

import "testing"

func TestParseHeader(t *testing.T) {
	if _, err := Parse("erDiagram\n"); err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if _, err := Parse("notER\n"); err == nil {
		t.Fatal("expected error for non-ER header")
	}
}

func TestParseRelationship(t *testing.T) {
	d, err := Parse(`erDiagram
CUSTOMER ||--o{ ORDER : places
`)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(d.Relationships) != 1 {
		t.Fatalf("rel count: %d", len(d.Relationships))
	}
	r := d.Relationships[0]
	if r.Left != "CUSTOMER" || r.Right != "ORDER" {
		t.Errorf("endpoints: %+v", r)
	}
	if r.LeftCardinality != CardExactlyOne || r.RightCardinality != CardZeroOrMore {
		t.Errorf("cards: %+v", r)
	}
	if !r.Identifying {
		t.Errorf("expected identifying: %+v", r)
	}
	if r.Label != "places" {
		t.Errorf("label: %q", r.Label)
	}
}

func TestParseAllCardinalities(t *testing.T) {
	src := `erDiagram
A ||--|| B : a
A |o--|o B : b
A }|--|{ B : c
A }o--o{ B : d
A ||..o{ B : e
`
	d, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(d.Relationships) != 5 {
		t.Fatalf("count: %d", len(d.Relationships))
	}
	if d.Relationships[4].Identifying {
		t.Errorf("dotted should be non-identifying")
	}
}

func TestParseEntityBlock(t *testing.T) {
	src := `erDiagram
CUSTOMER {
string name
int id PK
string email "primary contact"
}
`
	d, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(d.Entities) != 1 {
		t.Fatalf("entities: %d", len(d.Entities))
	}
	e := d.Entities[0]
	if len(e.Attributes) != 3 {
		t.Fatalf("attrs: %d", len(e.Attributes))
	}
	if e.Attributes[1].Key != KeyPrimary {
		t.Errorf("attr 1 key: %v", e.Attributes[1].Key)
	}
	if e.Attributes[2].Comment != "primary contact" {
		t.Errorf("comment: %q", e.Attributes[2].Comment)
	}
}
