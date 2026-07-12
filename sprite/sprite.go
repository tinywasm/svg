package sprite

import (
	"github.com/tinywasm/fmt"
	"github.com/tinywasm/json"
	"github.com/tinywasm/model"
	"github.com/tinywasm/svg"
)

// Node is internal typed markup for a symbol's body.
type Node string

// Path renders <path fill="currentColor" d="<d>"/>.
func Path(d string) Node {
	var b fmt.Builder
	b.WriteString("<path fill=\"currentColor\" d=\"")
	b.WriteString(d)
	b.WriteString("\"/>")
	return Node(b.String())
}

// Raw is an escape hatch for markup that Path does not express.
func Raw(s string) Node {
	return Node(s)
}

// Definition holds the data needed to build a <symbol>.
type Definition struct {
	Icon    svg.Icon
	ViewBox string
	Body    string
}

// Define creates a reusable icon definition.
func Define(icon svg.Icon, viewBox string, body ...Node) Definition {
	var b fmt.Builder
	for _, n := range body {
		b.WriteString(string(n))
	}
	return Definition{
		Icon:    icon,
		ViewBox: viewBox,
		Body:    b.String(),
	}
}

// Sprite holds SVG icon definitions for server-side sprite sheet injection.
type Sprite struct {
	icons []Definition
}

// NewSprite constructs a *Sprite from typed icons.
func NewSprite(icons ...Definition) *Sprite {
	return &Sprite{icons: icons}
}

// AddRaw adds a raw symbol to the sprite. Used for assetmin's raw .svg file/favicon path.
func (s *Sprite) AddRaw(id, content, viewBox string) {
	s.icons = append(s.icons, Definition{
		Icon:    svg.Icon(id),
		Body:    content,
		ViewBox: viewBox,
	})
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
	b.WriteString("<svg aria-hidden=\"true\" style=\"display:none\">")
	for _, e := range s.icons {
		b.WriteString("<symbol id=\"")
		b.WriteString(e.Icon.ID())
		b.WriteString("\" viewBox=\"")
		b.WriteString(e.ViewBox)
		b.WriteString("\">")
		b.WriteString(e.Body)
		b.WriteString("</symbol>")
	}
	b.WriteString("</svg>")
	return b.String()
}

// Encodable/Decodable implementation using github.com/tinywasm/json

func (s *Sprite) IsNil() bool {
	return s == nil
}

func (s *Sprite) EncodeFields(w model.FieldWriter) {}
func (s *Sprite) DecodeFields(r model.FieldReader) {}

func (s *Sprite) MarshalJSON() ([]byte, error) {
	if s == nil {
		return []byte("null"), nil
	}
	var out []byte
	err := json.Encode(s, &out)
	return out, err
}

func (s *Sprite) UnmarshalJSON(b []byte) error {
	return json.Decode(b, s)
}

// FielderSlice implementation for root array

func (s *Sprite) Len() int              { return len(s.icons) }
func (s *Sprite) At(i int) model.Fielder {
	return &iconFielder{icon: &s.icons[i]}
}
func (s *Sprite) Append() model.Fielder {
	s.icons = append(s.icons, Definition{})
	return &iconFielder{icon: &s.icons[len(s.icons)-1]}
}
func (s *Sprite) Schema() []model.Field { return nil }
func (s *Sprite) Pointers() []any       { return nil }

type iconFielder struct {
	icon *Definition
}

func (f *iconFielder) Schema() []model.Field { return nil }
func (f *iconFielder) Pointers() []any       { return nil }
func (f *iconFielder) IsNil() bool           { return f.icon == nil }

func (f *iconFielder) EncodeFields(w model.FieldWriter) {
	w.String("id", f.icon.Icon.ID())
	w.String("content", f.icon.Body)
	w.String("viewBox", f.icon.ViewBox)
}

func (f *iconFielder) DecodeFields(r model.FieldReader) {
	f.icon.Icon = svg.Icon(id(r.String("id")))
	f.icon.Body, _ = r.String("content")
	f.icon.ViewBox, _ = r.String("viewBox")
}

func id(s string, _ bool) string { return s }
