package mermaid

import "github.com/luo-studio/go-mermaid/displaylist"

// Dispatch helpers: each per-diagram-type package implements its own
// Parse + Layout. The top-level entry point can't import them
// directly because we want to keep mermaid.go's type detection and
// option types tightly coupled (and to avoid an import cycle if a
// per-type package wants to read mermaid.LayoutOptions).
//
// Phase 1 stubs return ErrNotImplemented; subsequent phases replace
// the stub bodies with calls into the real per-type packages.

func parseAndLayoutFlowchart(src string, opts LayoutOptions) (*displaylist.DisplayList, error) {
	return runFlowchart(src, opts)
}

func parseAndLayoutSequence(src string, opts LayoutOptions) (*displaylist.DisplayList, error) {
	return runSequence(src, opts)
}

func parseAndLayoutClass(src string, opts LayoutOptions) (*displaylist.DisplayList, error) {
	return runClass(src, opts)
}

func parseAndLayoutER(src string, opts LayoutOptions) (*displaylist.DisplayList, error) {
	return runER(src, opts)
}

func parseAndLayoutState(src string, opts LayoutOptions) (*displaylist.DisplayList, error) {
	return runState(src, opts)
}
