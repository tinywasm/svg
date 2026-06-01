package svg

import "github.com/tinywasm/dom"

// Svg builds an <svg> element.
func Svg(children ...any) *dom.Element {
	return dom.NewElement("svg").Add(children...)
}

// Use builds a <use> element for referencing sprite symbols.
func Use(children ...any) *dom.Element {
	return dom.NewElement("use").Add(children...)
}

// Icon creates an <svg><use href="#iconID"></svg> sprite reference.
// This is the canonical way to use an icon from the sprite sheet in Render().
//
// Example:
//
//	Icon("home", "nav-icon")
//	// → <svg aria-hidden='true' focusable='false' class='nav-icon'><use href='#home'></use></svg>
func Icon(iconID string, classes ...string) *dom.Element {
	el := Svg(
		Use().Attr("href", "#"+iconID),
	).Attr("aria-hidden", "true").Attr("focusable", "false")
	for _, c := range classes {
		el.Class(c)
	}
	return el
}
