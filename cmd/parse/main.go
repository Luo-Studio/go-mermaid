// Command parse reads a Mermaid diagram from stdin and writes its
// AST as JSON to stdout. Used for debugging and for piping into
// downstream tools.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	mermaid "github.com/luo-studio/go-mermaid"
	"github.com/luo-studio/go-mermaid/class"
	"github.com/luo-studio/go-mermaid/er"
	"github.com/luo-studio/go-mermaid/flowchart"
	"github.com/luo-studio/go-mermaid/sequence"
	"github.com/luo-studio/go-mermaid/state"
)

type wireOutput struct {
	Type string      `json:"type"`
	AST  interface{} `json:"ast"`
}

func main() {
	src, err := io.ReadAll(os.Stdin)
	if err != nil {
		fail(err)
	}
	t := mermaid.DetectDiagramType(string(src))
	var ast interface{}
	switch t {
	case "flowchart":
		ast, err = flowchart.Parse(string(src))
	case "sequence":
		ast, err = sequence.Parse(string(src))
	case "class":
		ast, err = class.Parse(string(src))
	case "er":
		ast, err = er.Parse(string(src))
	case "state":
		ast, err = state.Parse(string(src))
	default:
		fail(fmt.Errorf("mermaid: unrecognised diagram type"))
	}
	if err != nil {
		fail(err)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(wireOutput{Type: t, AST: ast}); err != nil {
		fail(err)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
