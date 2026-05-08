package fonts

import "fmt"

// Style identifies an embedded font variant.
type Style int

const (
	StyleRegular Style = iota
	StyleBold
	StyleItalic
)

// Bytes returns the raw TTF bytes for the requested style. The
// returned slice MUST NOT be mutated — it backs the embedded constant.
func Bytes(s Style) ([]byte, error) {
	switch s {
	case StyleRegular:
		return interRegular, nil
	case StyleBold:
		return interBold, nil
	case StyleItalic:
		return interItalic, nil
	default:
		return nil, fmt.Errorf("fonts: unknown style %d", s)
	}
}
