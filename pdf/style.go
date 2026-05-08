// Package mermaidpdf renders go-mermaid DisplayLists into fpdf
// documents. Imported as `mermaidpdf` to avoid stdlib `pdf`
// collisions.
package mermaidpdf

import "github.com/luo-studio/go-mermaid/displaylist"

// RoleStyle is the visual appearance of a single semantic Role.
//
// Color channels are 0-255. A negative value disables that channel:
// StrokeR < 0 = no stroke; FillR < 0 = transparent fill.
type RoleStyle struct {
	StrokeR, StrokeG, StrokeB float64
	StrokeWidth               float64
	DashPattern               []float64

	FillR, FillG, FillB float64

	TextR, TextG, TextB float64
	Font                string
	FontStyle           string
	FontSize            float64
}

// Style is the per-Role visual map. Default is used as fallback when
// a role has no explicit entry.
type Style struct {
	Roles   map[displaylist.Role]RoleStyle
	Default RoleStyle
}

// lookup returns the style for role, falling back to Default.
func (s Style) lookup(role displaylist.Role) RoleStyle {
	if rs, ok := s.Roles[role]; ok {
		return rs
	}
	return s.Default
}

// DefaultStyle returns a sensible black-on-white style.
func DefaultStyle() Style {
	body := RoleStyle{
		StrokeR: 30, StrokeG: 30, StrokeB: 30,
		StrokeWidth: 0.3,
		FillR:       -1, FillG: -1, FillB: -1,
		TextR: 0, TextG: 0, TextB: 0,
		Font: FontFamily, FontStyle: "", FontSize: 10,
	}
	bodyBold := body
	bodyBold.FontStyle = "B"
	muted := body
	muted.TextR, muted.TextG, muted.TextB = 100, 100, 100
	cluster := body
	cluster.StrokeR, cluster.StrokeG, cluster.StrokeB = 150, 150, 150
	cluster.FillR, cluster.FillG, cluster.FillB = 248, 248, 248
	clusterTitle := bodyBold
	activation := body
	activation.FillR, activation.FillG, activation.FillB = 220, 230, 245

	return Style{
		Default: body,
		Roles: map[displaylist.Role]RoleStyle{
			displaylist.RoleNode:             body,
			displaylist.RoleEdge:             body,
			displaylist.RoleEdgeLabel:        muted,
			displaylist.RoleSubgraph:         cluster,
			displaylist.RoleClusterTitle:     clusterTitle,
			displaylist.RoleActorBox:         body,
			displaylist.RoleActorTitle:       bodyBold,
			displaylist.RoleLifeline:         muted,
			displaylist.RoleActivation:       activation,
			displaylist.RoleMessageLabel:     body,
			displaylist.RoleNoteText:         body,
			displaylist.RoleSequenceNote:     cluster,
			displaylist.RoleLoopBlock:        cluster,
			displaylist.RoleAltBlock:         cluster,
			displaylist.RoleOptBlock:         cluster,
			displaylist.RoleParBlock:         cluster,
			displaylist.RoleCriticalBlock:    cluster,
			displaylist.RoleBreakBlock:       cluster,
			displaylist.RoleRectBlock:        cluster,
			displaylist.RoleClassBox:         body,
			displaylist.RoleClassMember:      body,
			displaylist.RoleClassAnnotation:  RoleStyle{StrokeR: -1, FillR: -1, TextR: 0, TextG: 0, TextB: 0, Font: FontFamily, FontStyle: "I", FontSize: 9},
			displaylist.RoleEntityBox:        body,
			displaylist.RoleEntityAttribute:  body,
			displaylist.RoleStateBox:         body,
			displaylist.RoleStateComposite:   cluster,
			displaylist.RolePseudostateStart: RoleStyle{StrokeR: 0, StrokeG: 0, StrokeB: 0, StrokeWidth: 0.3, FillR: 0, FillG: 0, FillB: 0, TextR: 0, TextG: 0, TextB: 0, Font: FontFamily, FontSize: 10},
			displaylist.RolePseudostateEnd:   RoleStyle{StrokeR: 0, StrokeG: 0, StrokeB: 0, StrokeWidth: 0.3, FillR: 0, FillG: 0, FillB: 0, TextR: 0, TextG: 0, TextB: 0, Font: FontFamily, FontSize: 10},
		},
	}
}
