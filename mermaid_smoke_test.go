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
		t.Fatalf("expected >=4 shapes, got %d", shapes)
	}
}

func TestPhase8AllDiagramTypes(t *testing.T) {
	cases := map[string]string{
		"flowchart": "flowchart TB\nA --> B\n",
		"sequence":  "sequenceDiagram\nparticipant A\nparticipant B\nA->>B: hi\n",
		"class":     "classDiagram\nclass A\nclass B\nA <|-- B\n",
		"er":        "erDiagram\nA ||--o{ B : has\n",
		"state":     "stateDiagram-v2\n[*] --> S\nS --> [*]\n",
	}
	for kind, src := range cases {
		t.Run(kind, func(t *testing.T) {
			dl, err := ParseAndLayout(src, LayoutOptions{})
			if err != nil {
				t.Fatalf("ParseAndLayout: %v", err)
			}
			if dl == nil || len(dl.Items) == 0 {
				t.Fatal("empty DisplayList")
			}
		})
	}
}

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
	if err != nil {
		t.Fatalf("ParseAndLayout: %v", err)
	}
	if dl.Width <= 0 {
		t.Fatal("empty bbox")
	}
	bullet, bullseye := 0, 0
	for _, it := range dl.Items {
		if s, ok := it.(displaylist.Shape); ok {
			if s.Kind == displaylist.ShapeKindStateBullet {
				bullet++
			}
			if s.Kind == displaylist.ShapeKindStateBullseye {
				bullseye++
			}
		}
	}
	if bullet < 1 || bullseye < 1 {
		t.Fatalf("expected >=1 bullet and >=1 bullseye, got %d/%d", bullet, bullseye)
	}
}

func TestPhase6Smoke(t *testing.T) {
	src := `erDiagram
CUSTOMER ||--o{ ORDER : places
CUSTOMER {
string name
int id PK
}
ORDER {
int id PK
int customer_id FK
}
`
	dl, err := ParseAndLayout(src, LayoutOptions{})
	if err != nil {
		t.Fatalf("ParseAndLayout: %v", err)
	}
	if dl.Width <= 0 {
		t.Fatalf("empty bbox")
	}
}

func TestPhase5Smoke(t *testing.T) {
	src := `classDiagram
class Animal {
+name : string
+eat() void
}
class Dog
class Cat
Animal <|-- Dog
Animal <|-- Cat
Dog *-- Bone
Cat o-- Toy
`
	dl, err := ParseAndLayout(src, LayoutOptions{})
	if err != nil {
		t.Fatalf("ParseAndLayout: %v", err)
	}
	classBoxes := 0
	for _, it := range dl.Items {
		if s, ok := it.(displaylist.Shape); ok && s.Role == displaylist.RoleClassBox {
			classBoxes++
		}
	}
	if classBoxes < 5 {
		t.Fatalf("expected >=5 class boxes, got %d", classBoxes)
	}
}

func TestPhase4Smoke(t *testing.T) {
	src := `sequenceDiagram
participant A as Alice
participant B as Bob
A->>B: Hi
B-->>A: Hello
loop Every minute
A->>B: ping
end
`
	dl, err := ParseAndLayout(src, LayoutOptions{})
	if err != nil {
		t.Fatalf("ParseAndLayout: %v", err)
	}
	if dl.Width <= 0 || dl.Height <= 0 {
		t.Fatalf("bbox: %vx%v", dl.Width, dl.Height)
	}
}

func TestPhase3Smoke(t *testing.T) {
	src := `flowchart TB
subgraph outer [Outer]
A --> B
subgraph inner [Inner]
C --> D
end
B --> C
end
A --> Z
`
	dl, err := ParseAndLayout(src, LayoutOptions{})
	if err != nil {
		t.Fatalf("ParseAndLayout: %v", err)
	}
	clusters := 0
	for _, it := range dl.Items {
		if _, ok := it.(displaylist.Cluster); ok {
			clusters++
		}
	}
	if clusters < 2 {
		t.Fatalf("expected >=2 clusters in nested subgraph, got %d", clusters)
	}
}
