package svg

import (
	"github.com/tinywasm/dom"
)

// Icon represents a reference to an SVG icon in the sprite sheet.
// It is a string type containing the icon's ID to keep WASM binary small.
type Icon string

// ID returns the symbol's ID.
func (i Icon) ID() string {
	return string(i)
}

// Render returns <svg aria-hidden='true' focusable='false' class=...><use href="#id"/></svg>.
func (i Icon) Render(classes ...string) *dom.Element {
	return IconLegacy(string(i), classes...)
}

// Svg builds an <svg> element.
func Svg(children ...dom.Component) *dom.Element {
	return dom.NewElement("svg").Child(children...)
}

// Use builds a <use> element for referencing sprite symbols.
func Use(children ...dom.Component) *dom.Element {
	return dom.NewElement("use").Child(children...)
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
