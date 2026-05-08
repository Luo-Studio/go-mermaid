package textutil

import (
	"reflect"
	"testing"
)

func TestCleanLabel(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"plain", "plain"},
		{"  spaced  ", "spaced"},
		{`"double quoted"`, "double quoted"},
		{`'single quoted'`, "single quoted"},
		{`""empty quoted layers""`, "empty quoted layers"},
		{`a<br/>b`, "a\nb"},
		{`a<br>b`, "a\nb"},
		{`a<br />b`, "a\nb"},
		{`a<BR/>b`, "a\nb"},
		{`A<br/>B<br/>C`, "A\nB\nC"},
		{`"with <br/> inside"`, "with \n inside"},
		{``, ``},
		{`""`, ``},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			if got := CleanLabel(tc.in); got != tc.want {
				t.Errorf("CleanLabel(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestSplitLabelLines(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"", []string{""}},
		{"single", []string{"single"}},
		{"a\nb", []string{"a", "b"}},
		{"a\nb\nc", []string{"a", "b", "c"}},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			got := SplitLabelLines(tc.in)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("SplitLabelLines(%q) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}
