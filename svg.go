package svg

import (
	"github.com/tinywasm/dom"
	"github.com/tinywasm/fmt"
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

// Icon is a typed and self-contained icon definition.
type Icon struct {
	id      string
	viewBox string
	body    string
}

// Define creates a reusable icon definition. viewBox is REQUIRED.
func Define(id, viewBox string, body ...Node) Icon {
	var b fmt.Builder
	for _, n := range body {
		b.WriteString(string(n))
	}
	return Icon{
		id:      id,
		viewBox: viewBox,
		body:    b.String(),
	}
}

// ID returns the symbol's ID.
func (i Icon) ID() string {
	return i.id
}

// Render returns <svg aria-hidden='true' focusable='false' class=...><use href="#id"/></svg>.
func (i Icon) Render(classes ...string) *dom.Element {
	return IconLegacy(i.id, classes...)
}

// Svg builds an <svg> element.
func Svg(children ...any) *dom.Element {
	return dom.NewElement("svg").Add(children...)
}

// Use builds a <use> element for referencing sprite symbols.
func Use(children ...any) *dom.Element {
	return dom.NewElement("use").Add(children...)
}

// IconLegacy creates an <svg><use href="#iconID"></svg> sprite reference.
//
// Deprecated: use (Icon).Render instead.
func IconLegacy(iconID string, classes ...string) *dom.Element {
	el := Svg(
		Use().Attr("href", "#"+iconID),
	).Attr("aria-hidden", "true").Attr("focusable", "false")
	for _, c := range classes {
		el.Class(c)
	}
	return el
}
