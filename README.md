# svg

SVG icon sprite API for TinyWasm — typed, zero-typo icon definitions.

## Overview

A typed API where **definition and reference share a single value**, viewBox is mandatory, and fill is infallible. A typo becomes a compilation error, not a blank icon at runtime.

Each project declares its own icons in `svg.go` — no central icon package. Icons are project-specific.

## Usage Pattern

### 1. Declare icons in `svg.go` (SSR only)

```go
//go:build !wasm

package platformd

import "github.com/tinywasm/svg"

// Package-level vars: each icon declared once with id + viewBox + paths.
var (
	iconHome = svg.Define("home", "0 0 576 512",
		svg.Path("M280 ..."), // multi-path example
		svg.Path("M120 ..."),
	)
	iconDoc = svg.Define("doc", "0 0 16 16",
		svg.Path("M7 ..."),
	)
	iconLogo = svg.Define("logo", "0 0 32 32",
		svg.Raw(`<circle cx="16" cy="16" r="16"/>`), // escape hatch for non-path shapes
	)
)

// IconSvg registers the sprite symbols that assetmin injects into <body>.
func (p *Platform) IconSvg() *svg.Sprite {
	return svg.NewSprite(iconHome, iconDoc, iconLogo)
}
```

### 2. Reference icons in Render

```go
// In your component's Render() — no build tag, compiled in WASM and !WASM:
link.Add(iconHome.Render(string(clsNavIcon)))
```

`iconHome` is the var from step 1. `Render` emits `<svg aria-hidden focusable=false class=...><use href="#home"/></svg>`. 
`clsNavIcon` is a typed `Class` var from tinywasm/css — converted to `string` because the Render signature accepts `...string`. 
If `clsNavIcon` is renamed in CSS, the compiler warns here too.

## API

### Define

```go
func Define(id, viewBox string, body ...node) Icon
```

Creates a reusable icon. `viewBox` is mandatory (no dangerous default that clips paths). `body` are one or more nodes (`Path` or `Raw`).

### Path

```go
func Path(d string) node
```

Renders `<path fill="currentColor" d="<d>"/>`. The fill is hardcoded — infallible, no one can forget it.

### Raw

```go
func Raw(s string) node
```

Escape hatch for markup that `Path` doesn't express: groups, circles, masks, gradients, etc. You decide where to use `fill="currentColor"` if you want theming.

### Icon.Render

```go
func (i Icon) Render(classes ...string) *dom.Element
```

Emits the `<svg><use>` reference with the correct id and classes.

### Icon.ID

```go
func (i Icon) ID() string
```

Returns the symbol id.

### NewSprite

```go
func NewSprite(icons ...Icon) *Sprite
```

Constructs a `*Sprite` from typed icons. The only way to build a sprite.

## Color: currentColor

`currentColor` **is not a color** — it's a reference to the CSS `color` property of the consuming element. 

The design requires the same icon to be colored differently in different contexts (e.g., secondary color in the nav, white when active). This works only with `currentColor` hardcoded in the symbol + color set by CSS at the use-site:

```go
// In the component's CSS:
Rule(clsNavIcon, Color(tokenColorSecondary))                     // normal
Rule(Selector(".pd-nav-active svg"), Color(tokenColorPrimary))   // active → white
```

The icon definition is **color-agnostic** (always `currentColor`); theming centralizes in CSS tokens at the use-site. No icon can be "non-themeable."

## Internal

- `Sprite` maintains its internal representation (`iconEntry{id, content, viewBox}`) and methods `String()`, `MarshalJSON`/`UnmarshalJSON`, `Merge()` **unchanged** — assetmin continues to inject `Sprite.String()` inline in `<body>` and serialize by JSON during SSR extraction. `Define` and `NewSprite` just construct that same data with compile-time safety.

- `Path(d)` produces exactly `<path fill="currentColor" d="...">` — the sprite result is byte-equivalent to manually-written markup.
