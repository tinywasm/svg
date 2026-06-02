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

