package sequence

import "testing"

func TestParseHeader(t *testing.T) {
	if _, err := Parse("sequenceDiagram\n"); err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if _, err := Parse("not-sequence\nA->>B: x\n"); err == nil {
		t.Fatal("expected error for non-sequence header")
	}
}

func TestParseParticipants(t *testing.T) {
	d, err := Parse("sequenceDiagram\nparticipant A as Alice\nactor B as Bob\n")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(d.Actors) != 2 {
		t.Fatalf("want 2 actors, got %d", len(d.Actors))
	}
	if d.Actors[0].ID != "A" || d.Actors[0].Label != "Alice" || d.Actors[0].Kind != ActorParticipant {
		t.Errorf("actor 0: %+v", d.Actors[0])
	}
	if d.Actors[1].Kind != ActorPerson {
		t.Errorf("actor 1 should be person: %+v", d.Actors[1])
	}
}

func TestParseMessages(t *testing.T) {
	d, err := Parse("sequenceDiagram\nA->>B: hi\nB-->>A: hey\nA-)B: ping\nB--)A: pong\n")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(d.Items) != 4 {
		t.Fatalf("want 4 items, got %d", len(d.Items))
	}
	wantArrows := []ArrowKind{ArrowSync, ArrowReply, ArrowOpen, ArrowOpenDashed}
	for i, want := range wantArrows {
		m := d.Items[i].(*Message)
		if m.Arrow != want {
			t.Errorf("msg %d: arrow %v want %v", i, m.Arrow, want)
		}
	}
}

func TestParseActivation(t *testing.T) {
	d, _ := Parse("sequenceDiagram\nA->>+B: do\nB-->>-A: done\n")
	if !d.Items[0].(*Message).Activate {
		t.Errorf("first msg should activate target")
	}
	if !d.Items[1].(*Message).Deactivate {
		t.Errorf("second msg should deactivate source")
	}
}

func TestParseNotes(t *testing.T) {
	d, _ := Parse("sequenceDiagram\nparticipant A\nparticipant B\nNote left of A: alone\nNote over A,B: shared\n")
	if len(d.Items) != 2 {
		t.Fatalf("want 2 items, got %d", len(d.Items))
	}
	left := d.Items[0].(*Note)
	if left.Side != NoteLeftOf || left.Text != "alone" {
		t.Errorf("left note: %+v", left)
	}
	over := d.Items[1].(*Note)
	if over.Side != NoteOver || len(over.Actors) != 2 {
		t.Errorf("over note: %+v", over)
	}
}

func TestParseBlocks(t *testing.T) {
	src := `sequenceDiagram
participant A
participant B
loop Every minute
A->>B: ping
end
alt Success
A->>B: ok
else Fail
A->>B: nope
end
`
	d, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(d.Items) != 2 {
		t.Fatalf("want 2 top-level blocks, got %d", len(d.Items))
	}
	loop := d.Items[0].(*Block)
	if loop.Type != BlockLoop || loop.Label != "Every minute" {
		t.Errorf("loop: %+v", loop)
	}
	if len(loop.Children) != 1 {
		t.Errorf("loop children: %d", len(loop.Children))
	}
	alt := d.Items[1].(*Block)
	if alt.Type != BlockAlt || len(alt.Else) != 1 {
		t.Errorf("alt: %+v", alt)
	}
	if alt.Else[0].Label != "Fail" {
		t.Errorf("alt else label: %q", alt.Else[0].Label)
	}
}

func TestParseUnclosedBlock(t *testing.T) {
	if _, err := Parse("sequenceDiagram\nloop X\nA->>B: hi\n"); err == nil {
		t.Fatal("expected error for unclosed loop")
	}
}
