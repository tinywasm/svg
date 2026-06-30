package svg

import "github.com/tinywasm/dom"

// Svg builds an <svg> element.
func Svg() *dom.Element {
	return dom.NewElement("svg")
}

// Use builds a <use> element for referencing sprite symbols.
func Use() *dom.Element {
	return dom.NewElement("use").NoCloseTag()
}
