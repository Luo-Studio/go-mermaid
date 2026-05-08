// Package fontmetrics provides the default Measurer used by the
// layout stage when the caller doesn't supply one. The measurer is
// backed by Inter font metrics extracted from the embedded TTFs.
package fontmetrics

import (
	"sync"

	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"

	"github.com/luo-studio/go-mermaid/fonts"
)

type faceStyle int

const (
	faceRegular faceStyle = iota
	faceBold
	faceItalic
)

type metrics struct {
	font       *sfnt.Font
	unitsPerEm int
	ascent     int
	descent    int
}

var (
	loadOnce sync.Once
	loadErr  error
	cache    [3]*metrics
)

func ensureLoaded() error {
	loadOnce.Do(func() {
		for i, style := range []faceStyle{faceRegular, faceBold, faceItalic} {
			data, err := fonts.Bytes(fontStyleFor(style))
			if err != nil {
				loadErr = err
				return
			}
			f, err := sfnt.Parse(data)
			if err != nil {
				loadErr = err
				return
			}
			var b sfnt.Buffer
			upm := int(f.UnitsPerEm())
			fm, err := f.Metrics(&b, fixed.I(upm), 0)
			if err != nil {
				loadErr = err
				return
			}
			cache[i] = &metrics{
				font:       f,
				unitsPerEm: upm,
				ascent:     fm.Ascent.Ceil(),
				descent:    fm.Descent.Ceil(),
			}
		}
	})
	return loadErr
}

// advance returns the advance width of r in font units. Falls back
// to half an em if the rune is not represented in the font.
func (m *metrics) advance(r rune) int {
	var b sfnt.Buffer
	idx, err := m.font.GlyphIndex(&b, r)
	if err != nil || idx == 0 {
		return m.unitsPerEm / 2
	}
	adv, err := m.font.GlyphAdvance(&b, idx, fixed.I(m.unitsPerEm), 0)
	if err != nil {
		return m.unitsPerEm / 2
	}
	return adv.Ceil()
}

func fontStyleFor(s faceStyle) fonts.Style {
	switch s {
	case faceBold:
		return fonts.StyleBold
	case faceItalic:
		return fonts.StyleItalic
	default:
		return fonts.StyleRegular
	}
}
