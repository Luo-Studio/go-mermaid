package er

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/luo-studio/go-mermaid/internal/textutil"
)

// Parse turns Mermaid erDiagram source into a Diagram.
func Parse(src string) (*Diagram, error) {
	lines := preprocess(strings.Split(src, "\n"))
	if len(lines) == 0 {
		return nil, fmt.Errorf("er: empty source")
	}
	if !strings.EqualFold(strings.TrimSpace(lines[0]), "erDiagram") {
		return nil, fmt.Errorf("er: line 1: expected `erDiagram`, got %q", lines[0])
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
	relationshipRE = regexp.MustCompile(`^([A-Za-z_]\w*)\s+([|o\}\{]{2})(--|\.\.)([|o\}\{]{2})\s+([A-Za-z_]\w*)(?:\s*:\s*(.*))?$`)
	entityStartRE  = regexp.MustCompile(`^([A-Za-z_]\w*)\s*\{\s*$`)
	entityCloseRE  = regexp.MustCompile(`^\}\s*$`)
	attrLineRE     = regexp.MustCompile(`^([\w\(\)]+)\s+([A-Za-z_]\w*)(?:\s+(PK|FK|UK))?(?:\s+"([^"]*)")?$`)
)

func parseBody(d *Diagram, lines []string) error {
	getOrAddEntity := func(id string) *Entity {
		for _, e := range d.Entities {
			if e.ID == id {
				return e
			}
		}
		e := &Entity{ID: id}
		d.Entities = append(d.Entities, e)
		return e
	}

	var openEntity *Entity
	for li, line := range lines {
		if openEntity != nil {
			if entityCloseRE.MatchString(line) {
				openEntity = nil
				continue
			}
			if m := attrLineRE.FindStringSubmatch(line); m != nil {
				attr := Attribute{
					Type:    m[1],
					Name:    m[2],
					Comment: textutil.CleanLabel(m[4]),
				}
				switch m[3] {
				case "PK":
					attr.Key = KeyPrimary
				case "FK":
					attr.Key = KeyForeign
				case "UK":
					attr.Key = KeyUnique
				}
				openEntity.Attributes = append(openEntity.Attributes, attr)
				continue
			}
			return fmt.Errorf("er: line %d: unrecognized attribute: %q", li+2, line)
		}

		switch {
		case entityStartRE.MatchString(line):
			m := entityStartRE.FindStringSubmatch(line)
			openEntity = getOrAddEntity(m[1])
		case relationshipRE.MatchString(line):
			m := relationshipRE.FindStringSubmatch(line)
			leftCard, ok := parseCard(m[2], true)
			if !ok {
				return fmt.Errorf("er: line %d: invalid left cardinality %q", li+2, m[2])
			}
			rightCard, ok := parseCard(m[4], false)
			if !ok {
				return fmt.Errorf("er: line %d: invalid right cardinality %q", li+2, m[4])
			}
			rel := &Relationship{
				Left:             m[1],
				Right:            m[5],
				LeftCardinality:  leftCard,
				RightCardinality: rightCard,
				Identifying:      m[3] == "--",
				Label:            textutil.CleanLabel(m[6]),
			}
			getOrAddEntity(m[1])
			getOrAddEntity(m[5])
			d.Relationships = append(d.Relationships, rel)
		default:
			return fmt.Errorf("er: line %d: unrecognized statement: %q", li+2, line)
		}
	}
	if openEntity != nil {
		return fmt.Errorf("er: unclosed entity %q", openEntity.ID)
	}
	return nil
}

// parseCard maps a 2-character cardinality token to a Cardinality.
// `leftSide` controls how the asymmetric tokens (e.g. `}|` vs `|{`)
// are interpreted — same semantics, different sides.
func parseCard(s string, leftSide bool) (Cardinality, bool) {
	switch s {
	case "||":
		return CardExactlyOne, true
	case "|o", "o|":
		return CardZeroOrOne, true
	case "}|", "|{":
		return CardOneOrMore, true
	case "}o", "o{":
		return CardZeroOrMore, true
	}
	return 0, false
}
