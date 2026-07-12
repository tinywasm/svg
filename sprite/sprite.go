package sprite

import (
	"errors"

	"github.com/tinywasm/fmt"
	tjson "github.com/tinywasm/json"
	"github.com/tinywasm/model"
	"github.com/tinywasm/svg"
)

// Node is internal typed markup for a symbol's body.
type Node string

// Path renders <path fill="currentColor" d="<d>"/>.
// The fill is fixed to currentColor to ensure the icon is always themeable.
func Path(d string) Node {
	var b fmt.Builder
	b.WriteString(`<path fill="currentColor" d="`)
	b.WriteString(d)
	b.WriteString(`"/>`)
	return Node(b.String())
}

// Raw is an escape hatch for markup that Path does not express (groups, circle, etc).
func Raw(s string) Node {
	return Node(s)
}

// IconDefinition is a typed and self-contained icon definition.
type IconDefinition struct {
	id      string
	viewBox string
	body    string
}

// Define creates a reusable icon definition. viewBox is REQUIRED.
func Define(id svg.Icon, viewBox string, body ...Node) IconDefinition {
	var b fmt.Builder
	for _, n := range body {
		b.WriteString(string(n))
	}
	return IconDefinition{
		id:      string(id),
		viewBox: viewBox,
		body:    b.String(),
	}
}

// Sprite holds SVG icon definitions for server-side sprite sheet injection.
type Sprite struct {
	icons []iconEntry
}

type iconEntry struct {
	id      string
	content string
	viewBox string
}

func (e *iconEntry) Schema() []model.Field {
	return []model.Field{
		{Name: "id", Type: model.Text()},
		{Name: "content", Type: model.Text()},
		{Name: "viewBox", Type: model.Text()},
	}
}

func (e *iconEntry) Pointers() []any {
	return []any{&e.id, &e.content, &e.viewBox}
}

func (e *iconEntry) IsNil() bool { return e == nil }

func (e *iconEntry) EncodeFields(w model.FieldWriter) {
	w.String("id", e.id)
	w.String("content", e.content)
	w.String("viewBox", e.viewBox)
}

func (e *iconEntry) DecodeFields(r model.FieldReader) {
	e.id, _ = r.String("id")
	e.content, _ = r.String("content")
	e.viewBox, _ = r.String("viewBox")
}

// New creates an empty Sprite.
func New() *Sprite {
	return &Sprite{}
}

// NewSprite constructs a *Sprite from typed icons.
func NewSprite(icons ...IconDefinition) *Sprite {
	s := New()
	for _, i := range icons {
		s.icons = append(s.icons, iconEntry{
			id:      i.id,
			content: i.body,
			viewBox: i.viewBox,
		})
	}
	return s
}

// Add registers an icon with defaults.
//
// Deprecated: use NewSprite with Define instead.
func (s *Sprite) Add(id, content string, viewBox ...string) *Sprite {
	vb := "0 0 16 16"
	if len(viewBox) > 0 && viewBox[0] != "" {
		vb = viewBox[0]
	}
	s.icons = append(s.icons, iconEntry{id: id, content: content, viewBox: vb})
	return s
}

// Merge adds all icons from other into s.
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

// SvgProvider is an optional capability for components to expose icons.
type SvgProvider interface {
	IconSvg() *Sprite
}

// model.FielderSlice implementation for Sprite

func (s *Sprite) Schema() []model.Field { return nil }
func (s *Sprite) Pointers() []any       { return nil }

func (s *Sprite) Len() int {
	if s == nil {
		return 0
	}
	return len(s.icons)
}

func (s *Sprite) At(i int) model.Fielder {
	return &s.icons[i]
}

func (s *Sprite) Append() model.Fielder {
	s.icons = append(s.icons, iconEntry{})
	return &s.icons[len(s.icons)-1]
}

// model.Encodable & model.Decodable implementation for Sprite

func (s *Sprite) IsNil() bool { return s == nil }

func (s *Sprite) EncodeFields(w model.FieldWriter) {
	aw := w.Array("", len(s.icons))
	for i := range s.icons {
		aw.Object(&s.icons[i])
	}
	aw.Close()
}

func (s *Sprite) DecodeFields(r model.FieldReader) {
	ar, ok := r.Array("")
	if !ok {
		return
	}
	n := ar.Len()
	s.icons = make([]iconEntry, n)
	for i := 0; i < n; i++ {
		ar.Object(i, &s.icons[i])
	}
}

// JSON Serialization

func (s *Sprite) MarshalJSON() ([]byte, error) {
	if s == nil {
		return []byte("null"), nil
	}
	var out string
	if err := tjson.Encode(s, &out); err != nil {
		return nil, err
	}
	return []byte(out), nil
}

func (s *Sprite) UnmarshalJSON(b []byte) error {
	if s == nil {
		return errors.New("cannot unmarshal into nil *Sprite")
	}
	return tjson.Decode(string(b), s)
}
