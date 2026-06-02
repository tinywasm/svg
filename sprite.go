package svg

import (
	"encoding/json"

	"github.com/tinywasm/fmt"
)

// Sprite holds SVG icon definitions for server-side sprite sheet injection.
// assetmin injects Sprite.String() inline at the top of <body>.
type Sprite struct {
	icons []iconEntry
}

type iconEntry struct {
	id      string
	content string // inner SVG content: <path fill="currentColor" d="..."/>
	viewBox string // e.g. "0 0 16 16"
}

// Merge adds all icons from other into s.
// Used by assetmin to accumulate icons from multiple components into one master sprite.
func (s *Sprite) Merge(other *Sprite) *Sprite {
	if other == nil {
		return s
	}
	if s == nil {
		return other
	}
	s.icons = append(s.icons, other.icons...)
	return s
}

// String renders the sprite as inline SVG with all <symbol> elements.
// Called by assetmin to inject at the top of <body>.
func (s *Sprite) String() string {
	if s == nil || len(s.icons) == 0 {
		return ""
	}
	var b fmt.Builder
	b.WriteString(`<svg aria-hidden="true" style="display:none">`)
	for _, e := range s.icons {
		b.WriteString(`<symbol id="`)
		b.WriteString(e.id)
		b.WriteString(`" viewBox="`)
		b.WriteString(e.viewBox)
		b.WriteString(`">`)
		b.WriteString(e.content)
		b.WriteString(`</symbol>`)
	}
	b.WriteString(`</svg>`)
	return b.String()
}

// jsonIcon is the serializable form of an icon (exported fields).
type jsonIcon struct {
	ID      string `json:"id"`
	Content string `json:"content"`
	ViewBox string `json:"viewBox"`
}

func (s *Sprite) MarshalJSON() ([]byte, error) {
	if s == nil {
		return []byte("null"), nil
	}
	out := make([]jsonIcon, len(s.icons))
	for i, e := range s.icons {
		out[i] = jsonIcon{ID: e.id, Content: e.content, ViewBox: e.viewBox}
	}
	return json.Marshal(out)
}

func (s *Sprite) UnmarshalJSON(b []byte) error {
	if s == nil {
		return fmt.Err("cannot unmarshal into nil *Sprite")
	}
	var in []jsonIcon
	if err := json.Unmarshal(b, &in); err != nil {
		return err
	}
	s.icons = make([]iconEntry, len(in))
	for i, e := range in {
		s.icons[i] = iconEntry{id: e.ID, content: e.Content, viewBox: e.ViewBox}
	}
	return nil
}
