package class

import (
	"fmt"
	"regexp"
	"strings"
)

// Parse turns Mermaid classDiagram source into a Diagram.
func Parse(src string) (*Diagram, error) {
	rawLines := strings.Split(src, "\n")
	cleaned := preprocess(rawLines)
	if len(cleaned) == 0 {
		return nil, fmt.Errorf("class: empty source")
	}
	if !strings.EqualFold(strings.TrimSpace(cleaned[0]), "classDiagram") {
		return nil, fmt.Errorf("class: line 1: expected `classDiagram`, got %q", cleaned[0])
	}
	d := &Diagram{}
	if err := parseBody(d, cleaned[1:]); err != nil {
		return nil, err
	}
	return d, nil
}

// preprocess strips inline comments, drops blank lines, but keeps line
// structure (no semicolon split, since `;` rarely appears in
// classDiagram source and would break member-list parsing).
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
	classDeclRE   = regexp.MustCompile(`^class\s+([A-Za-z_]\w*)(?:\s*\[([^\]]+)\])?\s*(\{)?\s*$`)
	classCloseRE  = regexp.MustCompile(`^\}\s*$`)
	annotationRE  = regexp.MustCompile(`^<<\s*(\w+)\s*>>(?:\s+([A-Za-z_]\w*))?\s*$`)
	memberInlineRE = regexp.MustCompile(`^([A-Za-z_]\w*)\s*:\s*(.+)$`)
	relationshipRE = regexp.MustCompile(`^([A-Za-z_]\w*)(?:\s+"([^"]+)")?\s+(<\|--|<\|\.\.|<\.\.\|>|--\|>|\|>--|<\|>--|\.\.\|>|\*--|--\*|o--|--o|<--|-->|<\.\.|\.\.>|--|\.\.)\s+(?:"([^"]+)"\s+)?([A-Za-z_]\w*)(?:\s*:\s*(.*))?$`)
	namespaceStartRE = regexp.MustCompile(`^namespace\s+([A-Za-z_]\w*)\s*\{?\s*$`)
)

func parseBody(d *Diagram, lines []string) error {
	classByID := map[string]int{}
	getOrAddClass := func(id string) *Class {
		if i, ok := classByID[id]; ok {
			return &d.Classes[i]
		}
		classByID[id] = len(d.Classes)
		d.Classes = append(d.Classes, Class{ID: id, Label: id})
		return &d.Classes[len(d.Classes)-1]
	}

	type frame struct {
		kind      string // "class" or "namespace"
		classID   string
		nsName    string
	}
	var stack []frame
	currentNS := func() string {
		for i := len(stack) - 1; i >= 0; i-- {
			if stack[i].kind == "namespace" {
				return stack[i].nsName
			}
		}
		return ""
	}

	addNamespace := func(name string) {
		for _, ns := range d.Namespaces {
			if ns.Name == name {
				return
			}
		}
		d.Namespaces = append(d.Namespaces, Namespace{Name: name})
	}
	assignToNamespace := func(classID, nsName string) {
		for i, ns := range d.Namespaces {
			if ns.Name == nsName {
				for _, id := range ns.ClassIDs {
					if id == classID {
						return
					}
				}
				d.Namespaces[i].ClassIDs = append(d.Namespaces[i].ClassIDs, classID)
				return
			}
		}
	}

	for li, line := range lines {
		// Inside an open class block: members until `}`.
		if len(stack) > 0 && stack[len(stack)-1].kind == "class" {
			if classCloseRE.MatchString(line) {
				stack = stack[:len(stack)-1]
				continue
			}
			classID := stack[len(stack)-1].classID
			c := getOrAddClass(classID)
			if m := annotationRE.FindStringSubmatch(line); m != nil && m[2] == "" {
				c.Annotation = "<<" + m[1] + ">>"
				continue
			}
			c.Members = append(c.Members, parseMember(line))
			continue
		}

		// Top level (or inside namespace).
		switch {
		case namespaceStartRE.MatchString(line):
			m := namespaceStartRE.FindStringSubmatch(line)
			addNamespace(m[1])
			stack = append(stack, frame{kind: "namespace", nsName: m[1]})
		case classCloseRE.MatchString(line):
			if len(stack) == 0 {
				return fmt.Errorf("class: line %d: unexpected `}`", li+2)
			}
			stack = stack[:len(stack)-1]
		case classDeclRE.MatchString(line):
			m := classDeclRE.FindStringSubmatch(line)
			id := m[1]
			label := m[2]
			c := getOrAddClass(id)
			if label != "" {
				c.Label = label
			}
			if ns := currentNS(); ns != "" {
				c.Namespace = ns
				assignToNamespace(id, ns)
			}
			if m[3] == "{" {
				stack = append(stack, frame{kind: "class", classID: id})
			}
		case annotationRE.MatchString(line):
			m := annotationRE.FindStringSubmatch(line)
			if m[2] != "" {
				c := getOrAddClass(m[2])
				c.Annotation = "<<" + m[1] + ">>"
			}
		case relationshipRE.MatchString(line):
			m := relationshipRE.FindStringSubmatch(line)
			rel := Relationship{
				From:     m[1],
				FromCard: m[2],
				ToCard:   m[4],
				To:       m[5],
				Label:    m[6],
			}
			rel.Kind, rel.Dashed = parseRelOp(m[3])
			// Some operators imply swap (e.g. `--*` is composition with
			// the diamond on the second class).
			if op := m[3]; op == "--*" || op == "--o" {
				rel.From, rel.To = rel.To, rel.From
				rel.FromCard, rel.ToCard = rel.ToCard, rel.FromCard
			}
			getOrAddClass(rel.From)
			getOrAddClass(rel.To)
			d.Relationships = append(d.Relationships, rel)
		case memberInlineRE.MatchString(line):
			m := memberInlineRE.FindStringSubmatch(line)
			c := getOrAddClass(m[1])
			c.Members = append(c.Members, parseMember(m[2]))
		default:
			return fmt.Errorf("class: line %d: unrecognized statement: %q", li+2, line)
		}
	}
	if len(stack) > 0 {
		return fmt.Errorf("class: %d unclosed block(s)", len(stack))
	}
	return nil
}

func parseRelOp(op string) (RelationKind, bool) {
	switch op {
	case "<|--":
		return RelInheritance, false
	case "<|..":
		return RelInheritance, true
	case "*--", "--*":
		return RelComposition, false
	case "o--", "--o":
		return RelAggregation, false
	case "-->":
		return RelAssociation, false
	case "<--":
		return RelAssociation, false
	case "..>":
		return RelDependency, true
	case "<..":
		return RelDependency, true
	case "..|>":
		return RelRealization, true
	case "--|>":
		return RelRealization, false
	case "--":
		return RelLink, false
	case "..":
		return RelLink, true
	}
	return RelAssociation, false
}

func parseMember(line string) Member {
	m := Member{}
	s := strings.TrimSpace(line)

	// Visibility prefix.
	if len(s) > 0 {
		switch s[0] {
		case '+':
			m.Visibility = VisPublic
			s = strings.TrimSpace(s[1:])
		case '-':
			m.Visibility = VisPrivate
			s = strings.TrimSpace(s[1:])
		case '#':
			m.Visibility = VisProtected
			s = strings.TrimSpace(s[1:])
		case '~':
			m.Visibility = VisPackage
			s = strings.TrimSpace(s[1:])
		}
	}

	// Static/abstract suffix marker: `$` static, `*` abstract (Mermaid).
	if strings.HasSuffix(s, "$") {
		m.IsStatic = true
		s = strings.TrimSpace(s[:len(s)-1])
	}
	if strings.HasSuffix(s, "*") && !strings.Contains(s, "(") {
		m.IsAbstract = true
		s = strings.TrimSpace(s[:len(s)-1])
	}

	// Method? Look for `(...)`.
	if pi := strings.Index(s, "("); pi >= 0 {
		m.IsMethod = true
		closeIdx := strings.Index(s[pi:], ")")
		if closeIdx >= 0 {
			m.Name = strings.TrimSpace(s[:pi])
			m.Args = s[pi : pi+closeIdx+1]
			rest := strings.TrimSpace(s[pi+closeIdx+1:])
			rest = strings.TrimSpace(strings.TrimPrefix(rest, ":"))
			if rest != "" {
				m.Type = rest
			}
			return m
		}
		m.Name = strings.TrimSpace(s)
		return m
	}

	// Attribute: optional "type name" (Mermaid uses "type name" too) or
	// "name : type".
	if ci := strings.Index(s, ":"); ci >= 0 {
		m.Name = strings.TrimSpace(s[:ci])
		m.Type = strings.TrimSpace(s[ci+1:])
		return m
	}
	parts := strings.Fields(s)
	if len(parts) == 2 {
		m.Type = parts[0]
		m.Name = parts[1]
	} else {
		m.Name = s
	}
	return m
}
