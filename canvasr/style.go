// Package mermaidcanvasr renders go-mermaid DisplayLists into raster
// images (PNG) and other vector formats supported by tdewolff/canvas.
// Imported as `mermaidcanvasr` to avoid collisions.
package mermaidcanvasr

import (
	"image/color"

	"github.com/tdewolff/canvas"

	"github.com/luo-studio/go-mermaid/displaylist"
	"github.com/luo-studio/go-mermaid/fonts"
)

// RoleStyle is the visual appearance of a single semantic Role.
type RoleStyle struct {
	Stroke      color.Color // nil = no stroke
	StrokeWidth float64
	DashPattern []float64

	Fill color.Color // nil = transparent

	TextColor  color.Color
	FontStyle  canvas.FontStyle // canvas.FontRegular / FontBold / FontItalic
	FontSize   float64          // pt
}

// Style maps DisplayList roles to RoleStyles.
type Style struct {
	FontFamily *canvas.FontFamily
	Roles      map[displaylist.Role]RoleStyle
	Default    RoleStyle
}

func (s Style) lookup(role displaylist.Role) RoleStyle {
	if rs, ok := s.Roles[role]; ok {
		return rs
	}
	return s.Default
}

// DefaultStyle returns a style backed by the embedded Inter family,
// black-on-white, with role-appropriate bold/italic for headers.
func DefaultStyle() (Style, error) {
	regBytes, err := fonts.Bytes(fonts.StyleRegular)
	if err != nil {
		return Style{}, err
	}
	boldBytes, err := fonts.Bytes(fonts.StyleBold)
	if err != nil {
		return Style{}, err
	}
	italicBytes, err := fonts.Bytes(fonts.StyleItalic)
	if err != nil {
		return Style{}, err
	}
	fam := canvas.NewFontFamily("Inter")
	if err := fam.LoadFont(regBytes, 0, canvas.FontRegular); err != nil {
		return Style{}, err
	}
	if err := fam.LoadFont(boldBytes, 0, canvas.FontBold); err != nil {
		return Style{}, err
	}
	if err := fam.LoadFont(italicBytes, 0, canvas.FontItalic); err != nil {
		return Style{}, err
	}

	body := RoleStyle{
		Stroke:      color.RGBA{R: 30, G: 30, B: 30, A: 255},
		StrokeWidth: 0.3,
		Fill:        nil,
		TextColor:   color.Black,
		FontStyle:   canvas.FontRegular,
		FontSize:    10,
	}
	bold := body
	bold.FontStyle = canvas.FontBold
	muted := body
	muted.TextColor = color.RGBA{R: 100, G: 100, B: 100, A: 255}
	cluster := body
	cluster.Fill = color.RGBA{R: 248, G: 248, B: 248, A: 255}
	cluster.Stroke = color.RGBA{R: 150, G: 150, B: 150, A: 255}
	activation := body
	activation.Fill = color.RGBA{R: 220, G: 230, B: 245, A: 255}
	pseudo := body
	pseudo.Stroke = color.Black
	pseudo.Fill = color.Black

	return Style{
		FontFamily: fam,
		Default:    body,
		Roles: map[displaylist.Role]RoleStyle{
			displaylist.RoleNode:             body,
			displaylist.RoleEdge:             body,
			displaylist.RoleEdgeLabel:        muted,
			displaylist.RoleSubgraph:         cluster,
			displaylist.RoleClusterTitle:     bold,
			displaylist.RoleActorBox:         body,
			displaylist.RoleActorTitle:       bold,
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
			displaylist.RoleClassAnnotation:  RoleStyle{Stroke: nil, Fill: nil, TextColor: color.Black, FontStyle: canvas.FontItalic, FontSize: 9},
			displaylist.RoleEntityBox:        body,
			displaylist.RoleEntityAttribute:  body,
			displaylist.RoleStateBox:         body,
			displaylist.RoleStateComposite:   cluster,
			displaylist.RolePseudostateStart: pseudo,
			displaylist.RolePseudostateEnd:   pseudo,
		},
	}, nil
}
