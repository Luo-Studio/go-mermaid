package fontmetrics

import (
	"github.com/luo-studio/go-mermaid/displaylist"
)

// DefaultMeasurer measures text using the embedded Inter metrics.
// Bold is selected for roles that are typically bold in Mermaid
// diagrams; italic is selected for class annotations.
type DefaultMeasurer struct {
	fontSize          float64
	regularLineHeight float64
	boldLineHeight    float64
	italicLineHeight  float64
}

// NewDefault returns a Measurer using the embedded Inter at the given
// point size. Panics if the embedded fonts can't be loaded.
func NewDefault(fontSize float64) *DefaultMeasurer {
	if err := ensureLoaded(); err != nil {
		panic("fontmetrics: cannot load embedded Inter: " + err.Error())
	}
	if fontSize <= 0 {
		fontSize = 14
	}
	d := &DefaultMeasurer{fontSize: fontSize}
	d.regularLineHeight = lineHeightFor(cache[faceRegular], fontSize)
	d.boldLineHeight = lineHeightFor(cache[faceBold], fontSize)
	d.italicLineHeight = lineHeightFor(cache[faceItalic], fontSize)
	return d
}

func lineHeightFor(m *metrics, fontSize float64) float64 {
	scale := fontSize / float64(m.unitsPerEm)
	return float64(m.ascent+m.descent) * scale
}

// Measure returns the rendered width and height of text in the
// Measurer's font at the requested role.
func (d *DefaultMeasurer) Measure(text string, role displaylist.Role) (w, h float64) {
	style := styleForRole(role)
	m := cache[style]
	scale := d.fontSize / float64(m.unitsPerEm)

	for _, r := range text {
		w += float64(m.advance(r)) * scale
	}
	switch style {
	case faceBold:
		h = d.boldLineHeight
	case faceItalic:
		h = d.italicLineHeight
	default:
		h = d.regularLineHeight
	}
	return
}

func styleForRole(role displaylist.Role) faceStyle {
	switch role {
	case displaylist.RoleClusterTitle,
		displaylist.RoleActorTitle,
		displaylist.RoleClassBox,
		displaylist.RoleEntityBox,
		displaylist.RoleStateBox:
		return faceBold
	case displaylist.RoleClassAnnotation:
		return faceItalic
	default:
		return faceRegular
	}
}
