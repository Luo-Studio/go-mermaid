package sequence

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/luo-studio/go-mermaid/internal/textutil"
)

// Parse turns Mermaid sequenceDiagram source into a Diagram.
func Parse(src string) (*Diagram, error) {
	lines := preprocess(src)
	if len(lines) == 0 {
		return nil, fmt.Errorf("sequence: empty source")
	}
	header := strings.TrimSpace(lines[0])
	if !strings.EqualFold(header, "sequenceDiagram") {
		return nil, fmt.Errorf("sequence: line 1: expected `sequenceDiagram`, got %q", lines[0])
	}
	d := &Diagram{}
	if err := parseBody(d, lines[1:]); err != nil {
		return nil, err
	}
	return d, nil
}

func preprocess(src string) []string {
	var out []string
	for _, line := range strings.Split(src, "\n") {
		if idx := strings.Index(line, "%%"); idx >= 0 {
			line = line[:idx]
		}
		s := strings.TrimSpace(line)
		if s == "" {
			continue
		}
		out = append(out, s)
	}
	return out
}

var (
	participantRE = regexp.MustCompile(`^(participant|actor)\s+([A-Za-z_]\w*)(?:\s+as\s+(.+))?$`)
	messageRE     = regexp.MustCompile(`^([A-Za-z_]\w*)\s*(-->>|->>|--\)|-\)|--x|-x|-->|->)\s*([+-]?)\s*([A-Za-z_]\w*)\s*:\s*(.*)$`)
	noteRE        = regexp.MustCompile(`(?i)^Note\s+(left of|right of|over)\s+([^:]+):\s*(.*)$`)
	blockStartRE  = regexp.MustCompile(`(?i)^(loop|alt|opt|par|critical|break|rect)(?:\s+(.*))?$`)
	elseRE        = regexp.MustCompile(`(?i)^else(?:\s+(.*))?$`)
	andRE         = regexp.MustCompile(`(?i)^and(?:\s+(.*))?$`)
	optionRE      = regexp.MustCompile(`(?i)^option(?:\s+(.*))?$`)
	endRE         = regexp.MustCompile(`(?i)^end\s*$`)
	activateRE    = regexp.MustCompile(`^activate\s+([A-Za-z_]\w*)\s*$`)
	deactivateRE  = regexp.MustCompile(`^deactivate\s+([A-Za-z_]\w*)\s*$`)
	autonumberRE  = regexp.MustCompile(`(?i)^autonumber.*$`)
)

func parseBody(d *Diagram, lines []string) error {
	type frame struct {
		block  *Block
		active *[]Item // pointer to the slice that new items get appended to
	}
	rootItems := &d.Items
	stack := []frame{{block: nil, active: rootItems}}

	ensureActor := func(id string) {
		for _, a := range d.Actors {
			if a.ID == id {
				return
			}
		}
		d.Actors = append(d.Actors, &Actor{ID: id, Label: id, Kind: ActorParticipant})
	}

	for li, line := range lines {
		current := &stack[len(stack)-1]
		switch {
		case autonumberRE.MatchString(line):
			// autonumber: not yet rendered, ignore
			continue
		case participantRE.MatchString(line):
			m := participantRE.FindStringSubmatch(line)
			id := m[2]
			label := textutil.CleanLabel(m[3])
			if label == "" {
				label = id
			}
			kind := ActorParticipant
			if strings.EqualFold(m[1], "actor") {
				kind = ActorPerson
			}
			// Replace existing if same ID, else append.
			found := false
			for _, a := range d.Actors {
				if a.ID == id {
					a.Label = label
					a.Kind = kind
					found = true
					break
				}
			}
			if !found {
				d.Actors = append(d.Actors, &Actor{ID: id, Label: label, Kind: kind})
			}
		case activateRE.MatchString(line):
			m := activateRE.FindStringSubmatch(line)
			ensureActor(m[1])
			*current.active = append(*current.active, &Activate{Actor: m[1]})
		case deactivateRE.MatchString(line):
			m := deactivateRE.FindStringSubmatch(line)
			ensureActor(m[1])
			*current.active = append(*current.active, &Deactivate{Actor: m[1]})
		case noteRE.MatchString(line):
			m := noteRE.FindStringSubmatch(line)
			side := parseNoteSide(m[1])
			actors := splitCommaList(m[2])
			for _, a := range actors {
				ensureActor(a)
			}
			*current.active = append(*current.active, &Note{
				Side:   side,
				Actors: actors,
				Text:   textutil.CleanLabel(m[3]),
			})
		case blockStartRE.MatchString(line):
			m := blockStartRE.FindStringSubmatch(line)
			bt := parseBlockType(m[1])
			block := &Block{Type: bt, Label: strings.TrimSpace(m[2])}
			*current.active = append(*current.active, block)
			stack = append(stack, frame{block: block, active: &block.Children})
		case elseRE.MatchString(line) || andRE.MatchString(line) || optionRE.MatchString(line):
			if current.block == nil {
				return fmt.Errorf("sequence: line %d: unexpected branch keyword outside a block", li+2)
			}
			var label string
			switch {
			case elseRE.MatchString(line):
				if m := elseRE.FindStringSubmatch(line); len(m) > 1 {
					label = strings.TrimSpace(m[1])
				}
			case andRE.MatchString(line):
				if m := andRE.FindStringSubmatch(line); len(m) > 1 {
					label = strings.TrimSpace(m[1])
				}
			case optionRE.MatchString(line):
				if m := optionRE.FindStringSubmatch(line); len(m) > 1 {
					label = strings.TrimSpace(m[1])
				}
			}
			branch := &Block{Type: current.block.Type, Label: label}
			current.block.Else = append(current.block.Else, branch)
			// Subsequent items go into the branch.
			stack[len(stack)-1].active = &branch.Children
		case endRE.MatchString(line):
			if len(stack) <= 1 {
				return fmt.Errorf("sequence: line %d: unexpected `end`", li+2)
			}
			stack = stack[:len(stack)-1]
		case messageRE.MatchString(line):
			m := messageRE.FindStringSubmatch(line)
			from := m[1]
			arrow := parseArrow(m[2])
			actMark := m[3]
			to := m[4]
			text := m[5]
			ensureActor(from)
			ensureActor(to)
			msg := &Message{
				From:  from,
				To:    to,
				Label: textutil.CleanLabel(text),
				Arrow: arrow,
			}
			switch actMark {
			case "+":
				msg.Activate = true
			case "-":
				msg.Deactivate = true
			}
			*current.active = append(*current.active, msg)
		default:
			return fmt.Errorf("sequence: line %d: unrecognized statement: %q", li+2, line)
		}
	}
	if len(stack) > 1 {
		return fmt.Errorf("sequence: %d unclosed block(s)", len(stack)-1)
	}
	return nil
}

func parseArrow(op string) ArrowKind {
	switch op {
	case "->>":
		return ArrowSync
	case "-->>":
		return ArrowReply
	case "-)":
		return ArrowOpen
	case "--)":
		return ArrowOpenDashed
	case "->":
		return ArrowSolid
	case "-->":
		return ArrowDashed
	case "-x":
		return ArrowCross
	case "--x":
		return ArrowDashedCross
	}
	return ArrowSync
}

func parseBlockType(s string) BlockType {
	switch strings.ToLower(s) {
	case "loop":
		return BlockLoop
	case "alt":
		return BlockAlt
	case "opt":
		return BlockOpt
	case "par":
		return BlockPar
	case "critical":
		return BlockCritical
	case "break":
		return BlockBreak
	case "rect":
		return BlockRect
	}
	return BlockLoop
}

func parseNoteSide(s string) NoteSide {
	switch strings.ToLower(s) {
	case "left of":
		return NoteLeftOf
	case "right of":
		return NoteRightOf
	case "over":
		return NoteOver
	}
	return NoteOver
}

func splitCommaList(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		t := strings.TrimSpace(p)
		if t != "" {
			out = append(out, t)
		}
	}
	return out
}
