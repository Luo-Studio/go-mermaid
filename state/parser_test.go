package state

import "testing"

func TestParseHeaderBoth(t *testing.T) {
	for _, h := range []string{"stateDiagram\n[*] --> A\n", "stateDiagram-v2\n[*] --> A\n"} {
		if _, err := Parse(h); err != nil {
			t.Errorf("Parse(%q): %v", h, err)
		}
	}
}

func TestParsePseudostates(t *testing.T) {
	d, err := Parse("stateDiagram-v2\n[*] --> A\nA --> [*]\n")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	starts, ends := 0, 0
	for _, s := range d.States {
		if s.IsStart {
			starts++
		}
		if s.IsEnd {
			ends++
		}
	}
	if starts != 1 || ends != 1 {
		t.Errorf("starts=%d ends=%d", starts, ends)
	}
}

func TestParseTransitionLabel(t *testing.T) {
	d, _ := Parse("stateDiagram\nA --> B : go\n")
	if len(d.Transitions) != 1 || d.Transitions[0].Label != "go" {
		t.Errorf("transitions: %+v", d.Transitions)
	}
}

func TestParseComposite(t *testing.T) {
	src := `stateDiagram-v2
[*] --> Idle
state Running {
[*] --> Working
Working --> Waiting : block
}
`
	d, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(d.Composites) != 1 {
		t.Fatalf("composites: %d", len(d.Composites))
	}
	c := d.Composites[0]
	if c.ID != "Running" {
		t.Errorf("composite id: %q", c.ID)
	}
	hasWorking := false
	for _, s := range c.StateIDs {
		if s == "Working" {
			hasWorking = true
		}
	}
	if !hasWorking {
		t.Errorf("composite states: %v", c.StateIDs)
	}
}

func TestParseStateAlias(t *testing.T) {
	d, _ := Parse(`stateDiagram-v2
state "Doing thing" as DT
[*] --> DT
`)
	for _, s := range d.States {
		if s.ID == "DT" {
			if s.Label != "Doing thing" {
				t.Errorf("alias label: %q", s.Label)
			}
			return
		}
	}
	t.Errorf("DT state not found in %+v", d.States)
}
