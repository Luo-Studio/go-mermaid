package state

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/luo-studio/go-mermaid/internal/textutil"
)

// Parse turns Mermaid stateDiagram source into a Diagram.
func Parse(src string) (*Diagram, error) {
	lines := preprocess(strings.Split(src, "\n"))
	if len(lines) == 0 {
		return nil, fmt.Errorf("state: empty source")
	}
	hdr := strings.TrimSpace(lines[0])
	if !strings.EqualFold(hdr, "stateDiagram") && !strings.EqualFold(hdr, "stateDiagram-v2") {
		return nil, fmt.Errorf("state: line 1: expected `stateDiagram` or `stateDiagram-v2`, got %q", lines[0])
	}
	d := &Diagram{}
	if err := parseBody(d, lines[1:]); err != nil {
		return nil, err
	}
	return d, nil
}

func preprocess(lines []string) []string {
	var out []string
	for _, raw := range lines {
		if idx := strings.Index(raw, "%%"); idx >= 0 {
			raw = raw[:idx]
		}
		s := strings.TrimSpace(raw)
		if s == "" {
			continue
		}
		out = append(out, s)
	}
	return out
}

var (
	stateAliasRE     = regexp.MustCompile(`^state\s+"([^"]+)"\s+as\s+([A-Za-z_]\w*)\s*$`)
	stateCompositeRE = regexp.MustCompile(`^state\s+([A-Za-z_]\w*)\s*\{\s*$`)
	stateDeclRE      = regexp.MustCompile(`^state\s+([A-Za-z_]\w*)\s*$`)
	closeRE          = regexp.MustCompile(`^\}\s*$`)
	transitionRE     = regexp.MustCompile(`^(\[\*\]|[A-Za-z_]\w*)\s+-->\s+(\[\*\]|[A-Za-z_]\w*)(?:\s*:\s*(.*))?$`)
	noteRE           = regexp.MustCompile(`(?i)^note\s+(left of|right of)\s+([A-Za-z_]\w*)\s*:\s*(.*)$`)
	noteOverRE       = regexp.MustCompile(`(?i)^note\s+over\s+([A-Za-z_]\w*)\s*:\s*(.*)$`)
)

func parseBody(d *Diagram, lines []string) error {
	stateIdx := map[string]int{}
	startIdxFor := map[string]int{} // composite ID → assigned start pseudostate index
	endIdxFor := map[string]int{}
	startCounter := 0
	endCounter := 0

	getOrAddState := func(id string, label string) int {
		label = textutil.CleanLabel(label)
		if i, ok := stateIdx[id]; ok {
			if d.States[i].Label == d.States[i].ID && label != "" {
				d.States[i].Label = label
			}
			return i
		}
		stateIdx[id] = len(d.States)
		actualLabel := label
		if actualLabel == "" {
			actualLabel = id
		}
		d.States = append(d.States, State{ID: id, Label: actualLabel})
		return stateIdx[id]
	}

	type frame struct {
		composite *Composite
	}
	stack := []frame{{nil}}

	currentCompositeID := func() string {
		if c := stack[len(stack)-1].composite; c != nil {
			return c.ID
		}
		return ""
	}

	addStateToCurrent := func(id string) {
		if c := stack[len(stack)-1].composite; c != nil {
			for _, s := range c.StateIDs {
				if s == id {
					return
				}
			}
			c.StateIDs = append(c.StateIDs, id)
		}
	}

	resolvePseudostate := func(token string, asStart bool) string {
		if token != "[*]" {
			return token
		}
		ctx := currentCompositeID()
		if asStart {
			if id, ok := startIdxFor[ctx]; ok {
				return d.States[id].ID
			}
			startCounter++
			id := fmt.Sprintf("__start_%d__", startCounter)
			d.States = append(d.States, State{ID: id, Label: "", IsStart: true})
			stateIdx[id] = len(d.States) - 1
			startIdxFor[ctx] = stateIdx[id]
			addStateToCurrent(id)
			return id
		}
		if id, ok := endIdxFor[ctx]; ok {
			return d.States[id].ID
		}
		endCounter++
		id := fmt.Sprintf("__end_%d__", endCounter)
		d.States = append(d.States, State{ID: id, Label: "", IsEnd: true})
		stateIdx[id] = len(d.States) - 1
		endIdxFor[ctx] = stateIdx[id]
		addStateToCurrent(id)
		return id
	}

	for li, line := range lines {
		switch {
		case stateAliasRE.MatchString(line):
			m := stateAliasRE.FindStringSubmatch(line)
			getOrAddState(m[2], m[1])
			addStateToCurrent(m[2])
		case stateCompositeRE.MatchString(line):
			m := stateCompositeRE.FindStringSubmatch(line)
			getOrAddState(m[1], m[1])
			addStateToCurrent(m[1])
			c := &Composite{ID: m[1], Label: m[1]}
			if parent := stack[len(stack)-1].composite; parent != nil {
				parent.Children = append(parent.Children, c)
			} else {
				d.Composites = append(d.Composites, c)
			}
			stack = append(stack, frame{composite: c})
		case stateDeclRE.MatchString(line):
			m := stateDeclRE.FindStringSubmatch(line)
			getOrAddState(m[1], "")
			addStateToCurrent(m[1])
		case closeRE.MatchString(line):
			if len(stack) <= 1 {
				return fmt.Errorf("state: line %d: unexpected `}`", li+2)
			}
			stack = stack[:len(stack)-1]
		case noteRE.MatchString(line):
			m := noteRE.FindStringSubmatch(line)
			i := getOrAddState(m[2], "")
			d.States[i].Note = m[3]
		case noteOverRE.MatchString(line):
			m := noteOverRE.FindStringSubmatch(line)
			i := getOrAddState(m[1], "")
			d.States[i].Note = m[2]
		case transitionRE.MatchString(line):
			m := transitionRE.FindStringSubmatch(line)
			from := resolvePseudostate(m[1], true)
			to := resolvePseudostate(m[2], false)
			if m[1] != "[*]" {
				getOrAddState(from, "")
				addStateToCurrent(from)
			}
			if m[2] != "[*]" {
				getOrAddState(to, "")
				addStateToCurrent(to)
			}
			d.Transitions = append(d.Transitions, Transition{From: from, To: to, Label: textutil.CleanLabel(m[3])})
		default:
			return fmt.Errorf("state: line %d: unrecognized statement: %q", li+2, line)
		}
	}
	if len(stack) > 1 {
		return fmt.Errorf("state: %d unclosed composite(s)", len(stack)-1)
	}
	// Composites are themselves states, but in the autog graph they
	// become clusters, not nodes. Remove their bare State entries from
	// the per-node graph by NOT adding them to autog later. Done in
	// layout.
	return nil
}
