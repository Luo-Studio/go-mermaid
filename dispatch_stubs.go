package mermaid

import (
	"github.com/luo-studio/go-mermaid/displaylist"
	"github.com/luo-studio/go-mermaid/flowchart"
)

func runFlowchart(src string, opts LayoutOptions) (*displaylist.DisplayList, error) {
	ast, err := flowchart.Parse(src)
	if err != nil {
		return nil, err
	}
	return flowchart.Layout(ast, opts), nil
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
