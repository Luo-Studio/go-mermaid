package mermaidcanvasr

import (
	"image/color"

	"github.com/tdewolff/canvas"

	"github.com/luo-studio/go-mermaid/displaylist"
	"github.com/luo-studio/go-mermaid/fonts"
	"github.com/luo-studio/go-mermaid/theme"
)

// StyleFromTheme returns a canvasr.Style backed by the given theme.
// The Inter family is loaded once and shared.
func StyleFromTheme(name string) (Style, error) {
	r, err := theme.Resolve(theme.Get(name))
	if err != nil {
		return Style{}, err
	}
	fam, err := loadInterFamily()
	if err != nil {
		return Style{}, err
	}
	return styleFromResolved(fam, r), nil
}

// MustStyleFromTheme is StyleFromTheme but panics on resolve failure.
func MustStyleFromTheme(name string) Style {
	s, err := StyleFromTheme(name)
	if err != nil {
		panic(err)
	}
	return s
}

func loadInterFamily() (*canvas.FontFamily, error) {
	regBytes, err := fonts.Bytes(fonts.StyleRegular)
	if err != nil {
		return nil, err
	}
	boldBytes, err := fonts.Bytes(fonts.StyleBold)
	if err != nil {
		return nil, err
	}
	italicBytes, err := fonts.Bytes(fonts.StyleItalic)
	if err != nil {
		return nil, err
	}
	fam := canvas.NewFontFamily("Inter")
	if err := fam.LoadFont(regBytes, 0, canvas.FontRegular); err != nil {
		return nil, err
	}
	if err := fam.LoadFont(boldBytes, 0, canvas.FontBold); err != nil {
		return nil, err
	}
	if err := fam.LoadFont(italicBytes, 0, canvas.FontItalic); err != nil {
		return nil, err
	}
	return fam, nil
}

func styleFromResolved(fam *canvas.FontFamily, r theme.Resolved) Style {
	asColor := func(c color.RGBA) color.Color { return c }
	body := RoleStyle{
		Stroke:      asColor(r.NodeStroke),
		StrokeWidth: 0.3,
		Fill:        asColor(r.NodeFill),
		TextColor:   asColor(r.Text),
		FontStyle:   canvas.FontRegular,
		FontSize:    10,
	}
	bodyBold := body
	bodyBold.FontStyle = canvas.FontBold

	edge := body
	edge.Stroke = asColor(r.Line)
	edge.Fill = nil
	edge.TextColor = asColor(r.Arrow)

	edgeLabel := body
	edgeLabel.Stroke = nil
	edgeLabel.Fill = nil
	edgeLabel.TextColor = asColor(r.TextSec)

	cluster := body
	cluster.Stroke = asColor(r.InnerStroke)
	cluster.Fill = asColor(r.GroupHeader)

	clusterTitle := bodyBold
	clusterTitle.Stroke = nil
	clusterTitle.Fill = nil

	muted := body
	muted.TextColor = asColor(r.TextSec)

	lifeline := body
	lifeline.Stroke = asColor(r.TextMuted)
	lifeline.Fill = nil

	activation := body
	activation.Stroke = asColor(r.InnerStroke)
	activation.Fill = asColor(r.KeyBadge)

	annotation := RoleStyle{
		Stroke: nil, Fill: nil,
		TextColor: asColor(r.TextFaint),
		FontStyle: canvas.FontItalic,
		FontSize:  9,
	}

	pseudo := body
	pseudo.Stroke = asColor(r.Text)
	pseudo.Fill = asColor(r.Text)

	return Style{
		FontFamily: fam,
		Default:    body,
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

// PageBackground returns the theme's background color.
func PageBackground(name string) color.Color {
	r, err := theme.Resolve(theme.Get(name))
	if err != nil {
		return color.White
	}
	return r.Bg
}
