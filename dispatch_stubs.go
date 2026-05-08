package mermaid

import "github.com/luo-studio/go-mermaid/displaylist"

// Phase 1 stubs. As later phases ship per-type packages
// (flowchart, sequence, class, er, state), these get replaced with
// thin shims over those packages' Parse + Layout pairs.

func runFlowchart(src string, opts LayoutOptions) (*displaylist.DisplayList, error) {
	_ = src
	_ = opts
	return nil, ErrNotImplemented
}

func runSequence(src string, opts LayoutOptions) (*displaylist.DisplayList, error) {
	_ = src
	_ = opts
	return nil, ErrNotImplemented
}

func runClass(src string, opts LayoutOptions) (*displaylist.DisplayList, error) {
	_ = src
	_ = opts
	return nil, ErrNotImplemented
}

func runER(src string, opts LayoutOptions) (*displaylist.DisplayList, error) {
	_ = src
	_ = opts
	return nil, ErrNotImplemented
}

func runState(src string, opts LayoutOptions) (*displaylist.DisplayList, error) {
	_ = src
	_ = opts
	return nil, ErrNotImplemented
}
