# svg
<img src="docs/img/badges.svg">

SVG icon sprite API for TinyWasm — typed, zero-typo icon definitions, split so
only the icon's *name* ever reaches the WASM binary.

## Overview

Two packages, two audiences:

- **`github.com/tinywasm/svg`** — the icon **reference**. `type Icon string` +
  `Render()`. Safe to import from ANY file, including one compiled into the
  browser bundle. This package never uses `//go:build` — it has nothing to
  hide from any target.
- **`github.com/tinywasm/svg/sprite`** — the icon **definition** (paths,
  viewBox, `<symbol>` markup, JSON serialization). Only backend code should
  import it: your own `svg.go` tagged `//go:build !wasm`, plus
  `tinywasm/ssr` (SSR extractor) and `assetmin`, which need it unconditionally.

Why split by package instead of a build tag inside this library: `tinywasm/ssr`
and `assetmin` are backend-only programs that need sprite construction at ALL
times — a tag *inside* this library can't express "always needed by this one
backend consumer." So the library stays untagged and minimal; **you, the
consumer, choose what to import and tag your own files accordingly.**

Each project declares its own icons — no central icon package. Icons are
project-specific.

## Usage Pattern

### 1. Declare the reference in shared code — no build tag

This is the part that reaches the WASM binary: just a name.

```go
// mycomponent.go — no build tag, compiles for browser AND backend
package mycomponent

import "github.com/tinywasm/svg"

const iconStar = svg.Icon("mycomponent-star") // just a string; nothing else ships to WASM
```

### 2. Draw the icon in YOUR OWN `svg.go` — `//go:build !wasm`

This is the part that never reaches the WASM binary. **You** add the tag —
this library does not and cannot add it for you.

```go
//go:build !wasm
package mycomponent

import "github.com/tinywasm/svg/sprite"

func (c *MyComponent) IconSvg() *sprite.Sprite {
	return sprite.NewSprite(
		sprite.Define(iconStar, "0 0 16 16",
			sprite.Path("M8 1l2 5h5l-4 3 1.5 5L8 11l-4.5 3L5 9 1 6h5z"),
		),
	)
}
```

`iconStar` is the constant from step 1 — same package, different file, one
identifier shared by definition and reference. A typo or rename breaks the
build in both places at once.

You never call `IconSvg()` yourself. The `tinywasm` framework does it
automatically as part of its build pipeline: `tinywasm/ssr` extracts it during
SSR and `assetmin` merges every component's sprite into one block and injects
it inline at the top of `<body>`. There is no `/assets/icons.svg` URL —
`href="#id"` always resolves without a network request.

For how that pipeline runs (dev server, hot-reload, installation), see
[`tinywasm/app`](https://github.com/tinywasm/app) — that's the framework
entry point, not this library.

### 3. Reference the icon in `Render()` — same shared file as step 1

```go
func (c *MyComponent) Render() *dom.Element {
	return dom.Div().Child(
		iconStar.Render(string(clsIcon)), // <svg aria-hidden focusable=false class=...><use href="#mycomponent-star"/></svg>
	)
}
```

`clsIcon` is a typed `Class` var from `tinywasm/css` — converted to `string`
because `Render` accepts `...string`. If `clsIcon` is renamed in CSS, the
compiler warns here too.

## API

### `svg` package

#### `type Icon string`

A typed reference to a sprite symbol. Declare as a package-level `const`.

#### `Icon.Render`

```go
func (i Icon) Render(classes ...string) *dom.Element
```

Emits the `<svg><use>` reference with the correct id and classes. **The only**
public way to build that markup — there is no `Svg()`/`Use()` builder.

### `svg/sprite` package

#### `Define`

```go
func Define(icon svg.Icon, viewBox string, body ...node) Symbol
```

Binds geometry to a shared `svg.Icon` reference. `viewBox` is mandatory (no
dangerous default that clips paths). `body` is one or more nodes (`Path` or
`Raw`).

#### `Path`

```go
func Path(d string) node
```

Renders `<path fill="currentColor" d="<d>"/>`. The fill is hardcoded —
infallible, no one can forget it.

#### `Raw`

```go
func Raw(s string) node
```

Escape hatch for markup that `Path` doesn't express: groups, circles, masks,
gradients, etc. You decide where to use `fill="currentColor"` if you want
theming.

#### `NewSprite`

```go
func NewSprite(symbols ...Symbol) *Sprite
```

Constructs a `*Sprite` from typed definitions. The only way to build a sprite.

## Color: currentColor

`currentColor` **is not a color** — it's a reference to the CSS `color`
property of the consuming element.

The design requires the same icon to be colored differently in different
contexts (e.g., secondary color in the nav, white when active). This works
only with `currentColor` hardcoded in the symbol + color set by CSS at the
use-site:

```go
// In the component's CSS:
Rule(clsNavIcon, Color(tokenColorSecondary))                     // normal
Rule(Selector(".pd-nav-active svg"), Color(tokenColorPrimary))   // active → white
```

The icon definition is **color-agnostic** (always `currentColor`); theming
centralizes in CSS tokens at the use-site. No icon can be "non-themeable."

## The one thing this library cannot enforce — read this

`svg/sprite` is a plain Go package with no build tag of its own — it compiles
fine for `GOOS=js GOARCH=wasm` too. Nothing stops you from importing it into a
file reachable by the browser build by mistake; that mistake **compiles
successfully** and silently ships path data + `encoding/json` into the WASM
binary. There is no compiler error to catch it.

This manual check is now automated during development via
`svg/watch.LeakGuard`, registered in `tinywasm/app`'s hot-reload (see
`app/docs/PLAN_SVG_LEAK_GUARD.md`) — the manual `go list -deps` command
remains documented as the CI/pre-publish fallback for environments without
the watcher running.

```bash
GOOS=js GOARCH=wasm go list -deps ./... | grep tinywasm/svg/sprite
```

Empty output = safe. Any output = some file without `//go:build !wasm` is
importing `svg/sprite` — find it and tag it (or move the import into your
tagged `svg.go`).

## Internal

- `Sprite` keeps its internal representation (`iconEntry{id, content,
  viewBox}`) and methods `String()`, `MarshalJSON`/`UnmarshalJSON`, `Merge()`
  unchanged — assetmin continues to inject `Sprite.String()` inline in
  `<body>` and serialize by JSON during SSR extraction. `Define` and
  `NewSprite` just construct that same data with compile-time safety.
- `Path(d)` produces exactly `<path fill="currentColor" d="...">` — the
  sprite result is byte-equivalent to manually-written markup.
