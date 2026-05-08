package flowchart

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/luo-studio/go-mermaid/internal/textutil"
)

// Parse turns Mermaid flowchart source into a Diagram.
func Parse(src string) (*Diagram, error) {
	lines := preprocess(src)
	if len(lines) == 0 {
		return nil, fmt.Errorf("flowchart: empty source")
	}
	d := &Diagram{
		Direction:  DirectionTB,
		NodeStyles: map[string]map[string]string{},
	}
	m := headerRE.FindStringSubmatch(lines[0])
	if m == nil {
		return nil, fmt.Errorf("flowchart: line 1: expected `flowchart` or `graph` header, got %q", lines[0])
	}
	switch strings.ToUpper(m[2]) {
	case "TB", "TD", "":
		d.Direction = DirectionTB
	case "BT":
		d.Direction = DirectionBT
	case "LR":
		d.Direction = DirectionLR
	case "RL":
		d.Direction = DirectionRL
	}
	if err := parseBody(d, lines[1:]); err != nil {
		return nil, err
	}
	return d, nil
}

// preprocess splits src by newlines and semicolons, trims segments,
// and drops empty / comment-only lines. Comments start with %%.
func preprocess(src string) []string {
	var out []string
	for _, line := range strings.Split(src, "\n") {
		// Strip in-line comments first (everything from `%%` to EOL).
		if idx := strings.Index(line, "%%"); idx >= 0 {
			line = line[:idx]
		}
		for _, frag := range strings.Split(line, ";") {
			s := strings.TrimSpace(frag)
			if s == "" {
				continue
			}
			out = append(out, s)
		}
	}
	return out
}

var (
	headerRE = regexp.MustCompile(`(?i)^(flowchart|graph)(?:\s+(TB|TD|BT|LR|RL))?\s*$`)

	classDefRE      = regexp.MustCompile(`^classDef\s+([A-Za-z_][\w-]*)\s+(.+)$`)
	classAssignRE   = regexp.MustCompile(`^class\s+([A-Za-z_][\w-]*(?:\s*,\s*[A-Za-z_][\w-]*)*)\s+([A-Za-z_][\w-]*)\s*$`)
	styleAssignRE   = regexp.MustCompile(`^style\s+([A-Za-z_][\w-]*)\s+(.+)$`)
	subgraphStartRE = regexp.MustCompile(`^subgraph\s+(.+)$`)
	subgraphEndRE   = regexp.MustCompile(`^end$`)
	directiveRE     = regexp.MustCompile(`^direction\s+(TB|TD|BT|LR|RL)\s*$`)

	// classShorthandRE matches the trailing `:::name` after a node decl.
	classShorthandRE = regexp.MustCompile(`^:::([A-Za-z_][\w-]*)`)

	// bareRE matches a bare node ID at the start of a string.
	bareRE = regexp.MustCompile(`^([\w-]+)`)
)

// shapePatterns maps source-syntax brackets to NodeShape. Order
// matters: multi-character delimiters appear first so e.g. `((text))`
// doesn't match `(text)`.
var shapePatterns = []struct {
	re    *regexp.Regexp
	shape NodeShape
}{
	{regexp.MustCompile(`^([\w-]+)\(\(\((.+?)\)\)\)`), ShapeDoubleCircle},
	{regexp.MustCompile(`^([\w-]+)\(\[(.+?)\]\)`), ShapeStadium},
	{regexp.MustCompile(`^([\w-]+)\(\((.+?)\)\)`), ShapeCircle},
	{regexp.MustCompile(`^([\w-]+)\[\[(.+?)\]\]`), ShapeSubroutine},
	{regexp.MustCompile(`^([\w-]+)\[\((.+?)\)\]`), ShapeCylinder},
	{regexp.MustCompile(`^([\w-]+)\[/(.+?)\\\]`), ShapeTrapezoid},
	{regexp.MustCompile(`^([\w-]+)\[\\(.+?)/\]`), ShapeTrapezoidAlt},
	{regexp.MustCompile(`^([\w-]+)\[/(.+?)/\]`), ShapeParallelogram},
	{regexp.MustCompile(`^([\w-]+)\[\\(.+?)\\\]`), ShapeParallelogramAlt},
	{regexp.MustCompile(`^([\w-]+)>(.+?)\]`), ShapeAsymmetric},
	{regexp.MustCompile(`^([\w-]+)\{\{(.+?)\}\}`), ShapeHexagon},
	{regexp.MustCompile(`^([\w-]+)\[(.+?)\]`), ShapeRect},
	{regexp.MustCompile(`^([\w-]+)\((.+?)\)`), ShapeRound},
	{regexp.MustCompile(`^([\w-]+)\{(.+?)\}`), ShapeDiamond},
}

// edgeOps is the set of plain (label-less) edge operators, listed
// longest-first so we never settle for a shorter prefix when a longer
// match exists.
type edgeOpSpec struct {
	op    string
	style EdgeStyle
	start bool
	end   bool
}

var edgeOps = []edgeOpSpec{
	{"<-.->", EdgeDotted, true, true},
	{"<-->", EdgeSolid, true, true},
	{"<==>", EdgeThick, true, true},
	{"-.->", EdgeDotted, false, true},
	{"<-.-", EdgeDotted, true, false},
	{"-->", EdgeSolid, false, true},
	{"<--", EdgeSolid, true, false},
	{"==>", EdgeThick, false, true},
	{"<==", EdgeThick, true, false},
	{"-.-", EdgeDotted, false, false},
	{"---", EdgeSolid, false, false},
	{"===", EdgeThick, false, false},
}

// inline-label edge patterns. Each captures: left-prefix-op, label,
// right-suffix-op. We then combine the two halves to determine the
// overall edge style and arrow ends.
//
// The patterns work on a windowed scan — we look for ` <opStart> ... <opEnd> `
// surrounded by spaces between two node specs.
var (
	inlineSolidRE  = regexp.MustCompile(`(<?-{2,})\s+(.+?)\s+(-{2,}>?)`)
	inlineThickRE  = regexp.MustCompile(`(<?={2,})\s+(.+?)\s+(={2,}>?)`)
	inlineDottedRE = regexp.MustCompile(`(<?-?\.+)\s+(.+?)\s+(\.+-?>?)`)
)

// parseBody walks the body lines after the header.
func parseBody(d *Diagram, lines []string) error {
	type frame struct {
		sg *Subgraph
	}
	stack := []frame{{nil}} // top-level frame has nil
	nodes := map[string]int{}

	addNodeMaybeMember := func(n Node) {
		if existing, ok := nodes[n.ID]; ok {
			cur := &d.Nodes[existing]
			if n.Label != "" && (cur.Label == "" || cur.Label == cur.ID) {
				cur.Label = n.Label
			}
			if cur.Shape == ShapeRect && n.Shape != ShapeRect {
				cur.Shape = n.Shape
			}
			cur.ClassNames = append(cur.ClassNames, n.ClassNames...)
			return
		}
		nodes[n.ID] = len(d.Nodes)
		d.Nodes = append(d.Nodes, n)
		if cur := stack[len(stack)-1].sg; cur != nil {
			cur.NodeIDs = append(cur.NodeIDs, n.ID)
		}
	}

	for li, line := range lines {
		switch {
		case directiveRE.MatchString(line):
			m := directiveRE.FindStringSubmatch(line)
			dir := DirectionTB
			switch strings.ToUpper(m[1]) {
			case "BT":
				dir = DirectionBT
			case "LR":
				dir = DirectionLR
			case "RL":
				dir = DirectionRL
			}
			if cur := stack[len(stack)-1].sg; cur != nil {
				cur.Direction = dir
			} else {
				d.Direction = dir
			}
		case classDefRE.MatchString(line):
			m := classDefRE.FindStringSubmatch(line)
			d.ClassDefs = append(d.ClassDefs, ClassDef{
				Name:       m[1],
				Properties: parseProps(m[2]),
			})
		case classAssignRE.MatchString(line):
			m := classAssignRE.FindStringSubmatch(line)
			ids := splitCommaList(m[1])
			for _, id := range ids {
				if idx, ok := nodes[id]; ok {
					d.Nodes[idx].ClassNames = append(d.Nodes[idx].ClassNames, m[2])
				} else {
					addNodeMaybeMember(Node{ID: id, Label: id, Shape: ShapeRect, ClassNames: []string{m[2]}})
				}
			}
		case styleAssignRE.MatchString(line):
			m := styleAssignRE.FindStringSubmatch(line)
			d.NodeStyles[m[1]] = parseProps(m[2])
		case subgraphStartRE.MatchString(line):
			m := subgraphStartRE.FindStringSubmatch(line)
			id, label := parseSubgraphHeader(m[1])
			sg := &Subgraph{ID: id, Label: label}
			if cur := stack[len(stack)-1].sg; cur != nil {
				cur.Children = append(cur.Children, sg)
			} else {
				d.Subgraphs = append(d.Subgraphs, sg)
			}
			stack = append(stack, frame{sg: sg})
		case subgraphEndRE.MatchString(line):
			if len(stack) <= 1 {
				return fmt.Errorf("flowchart: line %d: unexpected `end`", li+2)
			}
			stack = stack[:len(stack)-1]
		default:
			if err := parseEdgeOrNode(line, addNodeMaybeMember, func(e Edge) {
				d.Edges = append(d.Edges, e)
			}); err != nil {
				return fmt.Errorf("flowchart: line %d: %w", li+2, err)
			}
		}
	}
	if len(stack) > 1 {
		return fmt.Errorf("flowchart: %d unclosed subgraph(s)", len(stack)-1)
	}
	return nil
}

// parseEdgeOrNode tries to recognise an edge first; falls back to a
// bare node declaration.
func parseEdgeOrNode(line string, addNode func(Node), addEdge func(Edge)) error {
	if leftRaw, rightRaw, e, ok := matchEdge(line); ok {
		leftNode, err := parseNodeDecl(leftRaw)
		if err != nil {
			return err
		}
		rightNode, err := parseNodeDecl(rightRaw)
		if err != nil {
			return err
		}
		e.From = leftNode.ID
		e.To = rightNode.ID
		addNode(leftNode)
		addNode(rightNode)
		addEdge(e)
		return nil
	}
	n, err := parseNodeDecl(line)
	if err != nil {
		return err
	}
	addNode(n)
	return nil
}

// matchEdge looks for an edge operator (with or without inline label)
// inside line. On success returns left, right, and a partial Edge
// (Style/ArrowStart/ArrowEnd/Label set; From/To unset).
func matchEdge(line string) (left, right string, e Edge, ok bool) {
	// Try inline-label forms first (longest variants).
	if l, r, ed, found := tryInline(inlineSolidRE, EdgeSolid, line); found {
		return l, takePipeLabel(r, &ed), ed, true
	}
	if l, r, ed, found := tryInline(inlineThickRE, EdgeThick, line); found {
		return l, takePipeLabel(r, &ed), ed, true
	}
	if l, r, ed, found := tryInline(inlineDottedRE, EdgeDotted, line); found {
		return l, takePipeLabel(r, &ed), ed, true
	}
	// Plain operators, longest-first.
	for _, spec := range edgeOps {
		idx := strings.Index(line, spec.op)
		if idx < 0 {
			continue
		}
		left = strings.TrimSpace(line[:idx])
		right := strings.TrimSpace(line[idx+len(spec.op):])
		ed := Edge{Style: spec.style, ArrowStart: spec.start, ArrowEnd: spec.end}
		right = takePipeLabel(right, &ed)
		return left, right, ed, true
	}
	return "", "", Edge{}, false
}

// tryInline matches an inline-label edge using re. Returns left, the
// remaining-after-operator portion, and the partial Edge.
func tryInline(re *regexp.Regexp, base EdgeStyle, line string) (left, right string, e Edge, ok bool) {
	loc := re.FindStringSubmatchIndex(line)
	if loc == nil {
		return "", "", Edge{}, false
	}
	leftPart := strings.TrimSpace(line[:loc[0]])
	rightPart := strings.TrimSpace(line[loc[1]:])
	if leftPart == "" || rightPart == "" {
		return "", "", Edge{}, false
	}
	opStart := line[loc[2]:loc[3]]
	label := line[loc[4]:loc[5]]
	opEnd := line[loc[6]:loc[7]]

	e = Edge{Style: base, Label: cleanLabel(label)}
	e.ArrowStart = strings.HasPrefix(opStart, "<")
	e.ArrowEnd = strings.HasSuffix(opEnd, ">")
	return leftPart, rightPart, e, true
}

// takePipeLabel detects a `|label| ` prefix on right and, if present,
// extracts the label into e.Label and returns the trimmed remainder.
func takePipeLabel(right string, e *Edge) string {
	right = strings.TrimSpace(right)
	if !strings.HasPrefix(right, "|") {
		return right
	}
	end := strings.Index(right[1:], "|")
	if end < 0 {
		return right
	}
	e.Label = cleanLabel(right[1 : 1+end])
	return strings.TrimSpace(right[1+end+1:])
}

// parseNodeDecl parses a single node spec.
func parseNodeDecl(s string) (Node, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return Node{}, fmt.Errorf("empty node declaration")
	}
	for _, sp := range shapePatterns {
		if m := sp.re.FindStringSubmatch(s); m != nil {
			n := Node{ID: m[1], Label: cleanLabel(m[2]), Shape: sp.shape}
			rest := s[len(m[0]):]
			if cm := classShorthandRE.FindStringSubmatch(rest); cm != nil {
				n.ClassNames = append(n.ClassNames, cm[1])
			}
			return n, nil
		}
	}
	if m := bareRE.FindStringSubmatch(s); m != nil {
		n := Node{ID: m[1], Label: m[1], Shape: ShapeRect}
		rest := s[len(m[0]):]
		if cm := classShorthandRE.FindStringSubmatch(rest); cm != nil {
			n.ClassNames = append(n.ClassNames, cm[1])
		}
		return n, nil
	}
	return Node{}, fmt.Errorf("not a node declaration: %q", s)
}

func parseProps(s string) map[string]string {
	m := map[string]string{}
	for _, kv := range strings.Split(s, ",") {
		parts := strings.SplitN(kv, ":", 2)
		if len(parts) != 2 {
			continue
		}
		m[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
	}
	return m
}

func splitCommaList(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		out = append(out, strings.TrimSpace(p))
	}
	return out
}

func parseSubgraphHeader(rest string) (id, label string) {
	rest = strings.TrimSpace(rest)
	if i := strings.IndexAny(rest, "[\""); i >= 0 {
		idPart := strings.TrimSpace(rest[:i])
		labelPart := strings.TrimSpace(rest[i:])
		labelPart = strings.Trim(labelPart, "[]\"")
		labelPart = cleanLabel(labelPart)
		// `subgraph "Quoted Name"` (no separate ID): use the label as
		// both ID and display label so cluster lookups work.
		if idPart == "" {
			return labelPart, labelPart
		}
		return idPart, labelPart
	}
	return rest, cleanLabel(rest)
}

// cleanLabel is a thin wrapper over textutil.CleanLabel so existing
// call sites stay terse.
var cleanLabel = textutil.CleanLabel
