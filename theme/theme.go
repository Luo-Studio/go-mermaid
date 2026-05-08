// Package theme provides named color palettes for go-mermaid output.
//
// Ported from github.com/a-kaibu/mermaigo (yashikota fork). Each theme
// is a small DiagramColors record (Bg + Fg, optionally Line/Accent/
// Muted/Surface/Border). The Resolve helper expands those into the
// full set of derived colors emitters need (per-role text, line,
// arrow, fill, stroke, group header, inner stroke, key badge).
package theme

import (
	"fmt"
	"image/color"
	"sort"
	"strings"
)

// DiagramColors is the user-facing palette. Bg + Fg are required; the
// remaining fields override the derived defaults when set (otherwise
// they're computed from Bg + Fg via MixWeights).
type DiagramColors struct {
	Bg      string // background
	Fg      string // primary text / foreground
	Line    string // edge / connector color
	Accent  string // arrow heads, highlights
	Muted   string // secondary text, edge labels
	Surface string // node fill tint
	Border  string // node / group stroke
}

// MixWeights are the color-mix percentages mermaigo uses to derive
// secondary colors from Fg and Bg. Values are 0-100 and read as
// "X% Fg, (100-X)% Bg".
var MixWeights = struct {
	Text        int
	TextSec     int
	TextMuted   int
	TextFaint   int
	Line        int
	Arrow       int
	NodeFill    int
	NodeStroke  int
	GroupHeader int
	InnerStroke int
	KeyBadge    int
}{
	Text:        100,
	TextSec:     60,
	TextMuted:   40,
	TextFaint:   25,
	Line:        30,
	Arrow:       50,
	NodeFill:    3,
	NodeStroke:  20,
	GroupHeader: 5,
	InnerStroke: 12,
	KeyBadge:    10,
}

// Themes contains curated palettes mirroring mermaigo's selection.
var Themes = map[string]DiagramColors{
	"default": {
		Bg: "#FFFFFF", Fg: "#27272A",
	},
	"zinc-dark": {
		Bg: "#18181B", Fg: "#FAFAFA",
	},
	"tokyo-night": {
		Bg: "#1a1b26", Fg: "#a9b1d6",
		Line: "#3d59a1", Accent: "#7aa2f7", Muted: "#565f89",
	},
	"tokyo-night-storm": {
		Bg: "#24283b", Fg: "#a9b1d6",
		Line: "#3d59a1", Accent: "#7aa2f7", Muted: "#565f89",
	},
	"tokyo-night-light": {
		Bg: "#d5d6db", Fg: "#343b58",
		Line: "#34548a", Accent: "#34548a", Muted: "#9699a3",
	},
	"catppuccin-mocha": {
		Bg: "#1e1e2e", Fg: "#cdd6f4",
		Line: "#585b70", Accent: "#cba6f7", Muted: "#6c7086",
	},
	"catppuccin-latte": {
		Bg: "#eff1f5", Fg: "#4c4f69",
		Line: "#9ca0b0", Accent: "#8839ef", Muted: "#9ca0b0",
	},
	"nord": {
		Bg: "#2e3440", Fg: "#d8dee9",
		Line: "#4c566a", Accent: "#88c0d0", Muted: "#616e88",
	},
	"nord-light": {
		Bg: "#eceff4", Fg: "#2e3440",
		Line: "#aab1c0", Accent: "#5e81ac", Muted: "#7b88a1",
	},
	"dracula": {
		Bg: "#282a36", Fg: "#f8f8f2",
		Line: "#6272a4", Accent: "#bd93f9", Muted: "#6272a4",
	},
	"github-light": {
		Bg: "#ffffff", Fg: "#1f2328",
		Line: "#d1d9e0", Accent: "#0969da", Muted: "#59636e",
	},
	"github-dark": {
		Bg: "#0d1117", Fg: "#e6edf3",
		Line: "#3d444d", Accent: "#4493f8", Muted: "#9198a1",
	},
	"solarized-light": {
		Bg: "#fdf6e3", Fg: "#657b83",
		Line: "#93a1a1", Accent: "#268bd2", Muted: "#93a1a1",
	},
	"solarized-dark": {
		Bg: "#002b36", Fg: "#839496",
		Line: "#586e75", Accent: "#268bd2", Muted: "#586e75",
	},
	"one-dark": {
		Bg: "#282c34", Fg: "#abb2bf",
		Line: "#4b5263", Accent: "#c678dd", Muted: "#5c6370",
	},
}

// Names returns the available theme names in stable order.
func Names() []string {
	names := make([]string, 0, len(Themes))
	for n := range Themes {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// Get returns the named theme, or the default theme if name is unknown.
func Get(name string) DiagramColors {
	if t, ok := Themes[name]; ok {
		return t
	}
	return Themes["default"]
}

// Resolved is the fully expanded color set every emitter consumes.
// Every field is non-zero after Resolve.
type Resolved struct {
	Bg          color.RGBA
	Fg          color.RGBA
	Text        color.RGBA
	TextSec     color.RGBA
	TextMuted   color.RGBA
	TextFaint   color.RGBA
	Line        color.RGBA
	Arrow       color.RGBA
	NodeFill    color.RGBA
	NodeStroke  color.RGBA
	GroupFill   color.RGBA
	GroupHeader color.RGBA
	InnerStroke color.RGBA
	KeyBadge    color.RGBA
}

// Resolve expands a DiagramColors palette into concrete RGBA values
// for every derived role, mirroring mermaigo's CSS color-mix logic.
func Resolve(c DiagramColors) (Resolved, error) {
	bg, err := parseHex(c.Bg)
	if err != nil {
		return Resolved{}, fmt.Errorf("theme: bg %q: %w", c.Bg, err)
	}
	fg, err := parseHex(c.Fg)
	if err != nil {
		return Resolved{}, fmt.Errorf("theme: fg %q: %w", c.Fg, err)
	}

	pickOrMix := func(override string, weight int) (color.RGBA, error) {
		if override == "" {
			return mix(fg, bg, weight), nil
		}
		return parseHex(override)
	}

	r := Resolved{
		Bg:        bg,
		Fg:        fg,
		Text:      fg,
		GroupFill: bg,
	}
	r.TextSec, err = pickOrMix(c.Muted, MixWeights.TextSec)
	if err != nil {
		return Resolved{}, err
	}
	r.TextMuted, err = pickOrMix(c.Muted, MixWeights.TextMuted)
	if err != nil {
		return Resolved{}, err
	}
	r.TextFaint = mix(fg, bg, MixWeights.TextFaint)
	r.Line, err = pickOrMix(c.Line, MixWeights.Line)
	if err != nil {
		return Resolved{}, err
	}
	r.Arrow, err = pickOrMix(c.Accent, MixWeights.Arrow)
	if err != nil {
		return Resolved{}, err
	}
	r.NodeFill, err = pickOrMix(c.Surface, MixWeights.NodeFill)
	if err != nil {
		return Resolved{}, err
	}
	r.NodeStroke, err = pickOrMix(c.Border, MixWeights.NodeStroke)
	if err != nil {
		return Resolved{}, err
	}
	r.GroupHeader = mix(fg, bg, MixWeights.GroupHeader)
	r.InnerStroke = mix(fg, bg, MixWeights.InnerStroke)
	r.KeyBadge = mix(fg, bg, MixWeights.KeyBadge)
	return r, nil
}

// MustResolve is Resolve but panics on parse failure (used for the
// hardcoded named themes whose hex values are known good).
func MustResolve(c DiagramColors) Resolved {
	r, err := Resolve(c)
	if err != nil {
		panic(err)
	}
	return r
}

// parseHex accepts "#rrggbb" or "#rgb" (case-insensitive) and returns
// an opaque RGBA.
func parseHex(s string) (color.RGBA, error) {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "#") {
		return color.RGBA{}, fmt.Errorf("missing leading #")
	}
	hex := s[1:]
	switch len(hex) {
	case 3:
		var rgb [3]uint8
		for i := 0; i < 3; i++ {
			d, err := hexNibble(hex[i])
			if err != nil {
				return color.RGBA{}, err
			}
			rgb[i] = d*16 + d
		}
		return color.RGBA{R: rgb[0], G: rgb[1], B: rgb[2], A: 255}, nil
	case 6:
		var rgb [3]uint8
		for i := 0; i < 3; i++ {
			hi, err := hexNibble(hex[i*2])
			if err != nil {
				return color.RGBA{}, err
			}
			lo, err := hexNibble(hex[i*2+1])
			if err != nil {
				return color.RGBA{}, err
			}
			rgb[i] = hi*16 + lo
		}
		return color.RGBA{R: rgb[0], G: rgb[1], B: rgb[2], A: 255}, nil
	default:
		return color.RGBA{}, fmt.Errorf("expected #rgb or #rrggbb, got %q", s)
	}
}

func hexNibble(b byte) (uint8, error) {
	switch {
	case b >= '0' && b <= '9':
		return b - '0', nil
	case b >= 'a' && b <= 'f':
		return 10 + b - 'a', nil
	case b >= 'A' && b <= 'F':
		return 10 + b - 'A', nil
	}
	return 0, fmt.Errorf("invalid hex byte %q", b)
}

// mix interpolates a in toward b: result = a*pct/100 + b*(100-pct)/100
// per channel (8-bit sRGB, matches CSS color-mix(in srgb, ...) within
// rounding).
func mix(a, b color.RGBA, pct int) color.RGBA {
	if pct <= 0 {
		return b
	}
	if pct >= 100 {
		return a
	}
	w := float64(pct) / 100
	mixCh := func(x, y uint8) uint8 {
		v := float64(x)*w + float64(y)*(1-w)
		if v < 0 {
			v = 0
		}
		if v > 255 {
			v = 255
		}
		return uint8(v + 0.5)
	}
	return color.RGBA{
		R: mixCh(a.R, b.R),
		G: mixCh(a.G, b.G),
		B: mixCh(a.B, b.B),
		A: 255,
	}
}
