package svg

// SvgProvider is an optional capability: components that expose SVG icons
// for the global sprite sheet injected inline in <body> during SSR.
//
// Implement in a component's svg.go file (//go:build !wasm).
//
// Example in mycomponent/svg.go:
//
//	//go:build !wasm
//	package mycomponent
//	import "github.com/tinywasm/svg"
//
//	func (c *MyComponent) IconSvg() *svg.Sprite {
//	    return svg.New().
//	        Add("my-icon", `<path fill="currentColor" d="M0 0l16 16z"/>`)
//	}
type SvgProvider interface {
	IconSvg() *Sprite
}
