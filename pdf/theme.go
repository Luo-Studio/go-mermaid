package mermaidpdf

import (
	"image/color"

	"github.com/luo-studio/go-mermaid/displaylist"
	"github.com/luo-studio/go-mermaid/theme"
)

// StyleFromTheme returns a pdf.Style backed by the given theme. Roles
// map to the resolved colors per the standard convention used by the
// canvas rasterizer.
func StyleFromTheme(name string) (Style, error) {
	r, err := theme.Resolve(theme.Get(name))
	if err != nil {
		return Style{}, err
	}
	return styleFromResolved(r), nil
}

// MustStyleFromTheme is StyleFromTheme but panics on resolve failure.
// Safe for the bundled themes (whose hex values are known good).
func MustStyleFromTheme(name string) Style {
	s, err := StyleFromTheme(name)
	if err != nil {
		panic(err)
	}
	return s
}

func styleFromResolved(r theme.Resolved) Style {
	body := RoleStyle{
		StrokeR: float64(r.NodeStroke.R), StrokeG: float64(r.NodeStroke.G), StrokeB: float64(r.NodeStroke.B),
		StrokeWidth: 0.3,
		FillR:       float64(r.NodeFill.R), FillG: float64(r.NodeFill.G), FillB: float64(r.NodeFill.B),
		TextR: float64(r.Text.R), TextG: float64(r.Text.G), TextB: float64(r.Text.B),
		Font: FontFamily,
		FontSize: 10,
	}
	bodyBold := body
	bodyBold.FontStyle = "B"

	edge := body
	edge.StrokeR, edge.StrokeG, edge.StrokeB = float64(r.Line.R), float64(r.Line.G), float64(r.Line.B)
	edge.FillR, edge.FillG, edge.FillB = -1, -1, -1
	edge.TextR, edge.TextG, edge.TextB = float64(r.Arrow.R), float64(r.Arrow.G), float64(r.Arrow.B)

	edgeLabel := body
	edgeLabel.TextR, edgeLabel.TextG, edgeLabel.TextB = float64(r.TextSec.R), float64(r.TextSec.G), float64(r.TextSec.B)
	edgeLabel.StrokeR = -1
	edgeLabel.FillR = -1

	cluster := body
	cluster.StrokeR, cluster.StrokeG, cluster.StrokeB = float64(r.InnerStroke.R), float64(r.InnerStroke.G), float64(r.InnerStroke.B)
	cluster.FillR, cluster.FillG, cluster.FillB = float64(r.GroupHeader.R), float64(r.GroupHeader.G), float64(r.GroupHeader.B)

	clusterTitle := bodyBold
	clusterTitle.StrokeR = -1
	clusterTitle.FillR = -1

	muted := body
	muted.TextR, muted.TextG, muted.TextB = float64(r.TextSec.R), float64(r.TextSec.G), float64(r.TextSec.B)

	lifeline := body
	lifeline.StrokeR, lifeline.StrokeG, lifeline.StrokeB = float64(r.TextMuted.R), float64(r.TextMuted.G), float64(r.TextMuted.B)
	lifeline.FillR = -1

	activation := body
	activation.FillR, activation.FillG, activation.FillB = float64(r.KeyBadge.R), float64(r.KeyBadge.G), float64(r.KeyBadge.B)
	activation.StrokeR, activation.StrokeG, activation.StrokeB = float64(r.InnerStroke.R), float64(r.InnerStroke.G), float64(r.InnerStroke.B)

	annotation := RoleStyle{
		StrokeR: -1, StrokeG: -1, StrokeB: -1,
		FillR: -1, FillG: -1, FillB: -1,
		TextR: float64(r.TextFaint.R), TextG: float64(r.TextFaint.G), TextB: float64(r.TextFaint.B),
		Font: FontFamily, FontStyle: "I", FontSize: 9,
	}

	pseudo := body
	pseudo.StrokeR, pseudo.StrokeG, pseudo.StrokeB = float64(r.Text.R), float64(r.Text.G), float64(r.Text.B)
	pseudo.FillR, pseudo.FillG, pseudo.FillB = float64(r.Text.R), float64(r.Text.G), float64(r.Text.B)

	return Style{
		Default: body,
		Roles: map[displaylist.Role]RoleStyle{
			displaylist.RoleNode:             body,
			displaylist.RoleEdge:             edge,
			displaylist.RoleEdgeLabel:        edgeLabel,
			displaylist.RoleSubgraph:         cluster,
			displaylist.RoleClusterTitle:     clusterTitle,
			displaylist.RoleActorBox:         body,
			displaylist.RoleActorTitle:       bodyBold,
			displaylist.RoleLifeline:         lifeline,
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
			displaylist.RoleClassMember:      muted,
			displaylist.RoleClassAnnotation:  annotation,
			displaylist.RoleEntityBox:        body,
			displaylist.RoleEntityAttribute:  muted,
			displaylist.RoleStateBox:         body,
			displaylist.RoleStateComposite:   cluster,
			displaylist.RolePseudostateStart: pseudo,
			displaylist.RolePseudostateEnd:   pseudo,
		},
	}
}

// PageBackground returns the (R, G, B) of the theme's background color.
// Useful when the caller wants to fill the PDF page with the theme bg
// before drawing the diagram.
func PageBackground(name string) (r, g, b int) {
	rs, err := theme.Resolve(theme.Get(name))
	if err != nil {
		return 255, 255, 255
	}
	return int(rs.Bg.R), int(rs.Bg.G), int(rs.Bg.B)
}

// avoid unused import when later edits remove color usage
var _ = color.RGBA{}
