package theme

import (
	"image/color"
	"testing"
)

func TestNamesIncludesAll(t *testing.T) {
	names := Names()
	if len(names) < 14 {
		t.Fatalf("expected >=14 themes, got %d", len(names))
	}
	for _, want := range []string{"default", "dracula", "tokyo-night", "github-dark", "solarized-dark"} {
		found := false
		for _, n := range names {
			if n == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing theme: %q", want)
		}
	}
}

func TestGetUnknownReturnsDefault(t *testing.T) {
	got := Get("does-not-exist")
	if got.Bg != "#FFFFFF" {
		t.Errorf("unknown should fall back to default (white bg), got %q", got.Bg)
	}
}

func TestResolveDarkTheme(t *testing.T) {
	r, err := Resolve(Get("dracula"))
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if r.Bg.R == 0 && r.Bg.G == 0 && r.Bg.B == 0 {
		t.Errorf("dracula bg should not be pure black: %+v", r.Bg)
	}
	// Dracula bg is #282a36 — 0x28=40, 0x2a=42, 0x36=54
	if r.Bg.R != 0x28 || r.Bg.G != 0x2a || r.Bg.B != 0x36 {
		t.Errorf("dracula bg parse: got %+v", r.Bg)
	}
	if r.Fg.R != 0xf8 {
		t.Errorf("dracula fg parse: got %+v", r.Fg)
	}
	// Line is provided explicitly (#6272a4); should not be the mix.
	if r.Line.R != 0x62 || r.Line.G != 0x72 || r.Line.B != 0xa4 {
		t.Errorf("dracula line override: got %+v", r.Line)
	}
}

func TestResolveDerivesWhenOmitted(t *testing.T) {
	// zinc-dark sets only Bg + Fg; derived colors must mix Fg into Bg.
	r, err := Resolve(Get("zinc-dark"))
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	// Line = 30% fg + 70% bg. zinc-dark fg = #FAFAFA, bg = #18181B.
	// 30% * 0xFA + 70% * 0x18 = 75 + 16.8 = ~92 = 0x5c (rounding: 75+17=92)
	wantRf := float64(0xFA)*0.30 + float64(0x18)*0.70 + 0.5
	wantR := uint8(wantRf)
	if abs(int(r.Line.R)-int(wantR)) > 1 {
		t.Errorf("zinc-dark Line.R: got %d want ~%d", r.Line.R, wantR)
	}
	if r.Line == (color.RGBA{}) {
		t.Errorf("derived Line should be non-zero")
	}
}

func TestParseHex(t *testing.T) {
	cases := map[string]color.RGBA{
		"#000":    {R: 0, G: 0, B: 0, A: 255},
		"#fff":    {R: 0xff, G: 0xff, B: 0xff, A: 255},
		"#FFFFFF": {R: 0xff, G: 0xff, B: 0xff, A: 255},
		"#1a2b3c": {R: 0x1a, G: 0x2b, B: 0x3c, A: 255},
	}
	for s, want := range cases {
		got, err := parseHex(s)
		if err != nil {
			t.Errorf("parseHex(%q): %v", s, err)
			continue
		}
		if got != want {
			t.Errorf("parseHex(%q) = %+v, want %+v", s, got, want)
		}
	}
	for _, bad := range []string{"", "FFF", "#GG0000", "#1234"} {
		if _, err := parseHex(bad); err == nil {
			t.Errorf("parseHex(%q) should fail", bad)
		}
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
