package displaylist

import (
	"encoding/json"
	"fmt"
)

type wireItem struct {
	Kind string          `json:"kind"`
	Body json.RawMessage `json:"body"`
}

type wireDisplayList struct {
	Width  float64    `json:"width"`
	Height float64    `json:"height"`
	Items  []wireItem `json:"items"`
}

// MarshalJSON encodes a DisplayList with a per-item kind discriminator
// so the closed sum type round-trips through JSON.
func (dl DisplayList) MarshalJSON() ([]byte, error) {
	w := wireDisplayList{Width: dl.Width, Height: dl.Height}
	for _, it := range dl.Items {
		body, err := json.Marshal(it)
		if err != nil {
			return nil, fmt.Errorf("displaylist: marshal %s item: %w", it.itemKind(), err)
		}
		w.Items = append(w.Items, wireItem{Kind: it.itemKind(), Body: body})
	}
	return json.Marshal(w)
}

// UnmarshalJSON decodes the discriminated form back into typed Items.
func (dl *DisplayList) UnmarshalJSON(data []byte) error {
	var w wireDisplayList
	if err := json.Unmarshal(data, &w); err != nil {
		return err
	}
	dl.Width = w.Width
	dl.Height = w.Height
	dl.Items = nil
	for i, wi := range w.Items {
		var it Item
		switch wi.Kind {
		case "shape":
			var s Shape
			if err := json.Unmarshal(wi.Body, &s); err != nil {
				return fmt.Errorf("displaylist: item %d shape body: %w", i, err)
			}
			it = s
		case "edge":
			var e Edge
			if err := json.Unmarshal(wi.Body, &e); err != nil {
				return fmt.Errorf("displaylist: item %d edge body: %w", i, err)
			}
			it = e
		case "text":
			var x Text
			if err := json.Unmarshal(wi.Body, &x); err != nil {
				return fmt.Errorf("displaylist: item %d text body: %w", i, err)
			}
			it = x
		case "cluster":
			var c Cluster
			if err := json.Unmarshal(wi.Body, &c); err != nil {
				return fmt.Errorf("displaylist: item %d cluster body: %w", i, err)
			}
			it = c
		case "marker":
			var m Marker
			if err := json.Unmarshal(wi.Body, &m); err != nil {
				return fmt.Errorf("displaylist: item %d marker body: %w", i, err)
			}
			it = m
		default:
			return fmt.Errorf("displaylist: unknown item kind %q at index %d", wi.Kind, i)
		}
		dl.Items = append(dl.Items, it)
	}
	return nil
}
