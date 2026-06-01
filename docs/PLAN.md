# PLAN: tinywasm/svg — SVG Builders + Sprite System

## Repositorio
`github.com/tinywasm/svg` — path local: `tinywasm/svg/`  
Estado actual: go.mod `module github.com/cdvelop/svg`, `svg.go` stub (`type Svg struct{}`)

## Dependencias de ejecución
```bash
go install github.com/tinywasm/devflow/cmd/gotest@latest
```

## Prerequisito
Ejecutar **después** de que `tinywasm/dom` haya publicado su versión con `NewElement()` y `NoCloseTag()`.

---

## Objetivo

`tinywasm/svg` provee solo lo que se usa actualmente en el ecosistema tinywasm:

1. **Builders SVG mínimos** — `Svg()`, `Use()`, `Icon()` (los únicos en uso)
2. **Sistema de Sprite** — `*Sprite` con `Add()`, `Merge()`, `String()`, para reemplazar `map[string]string`

**Separación de responsabilidad con assetmin:**
- `tinywasm/svg` → sabe **cómo construir** el sprite HTML (`<symbol>`, `viewBox`, `String()`)
- `assetmin` → sabe **cuándo y dónde inyectar** el sprite en el HTML body

assetmin mantiene internamente un `*svg.Sprite` master, llama `master.Merge(comp.IconSvg())` por cada componente, e inyecta `master.String()`.

---

## go.mod

```
module github.com/tinywasm/svg

go 1.25

require (
    github.com/tinywasm/dom v<nueva-version>
)
```

---

## Paso 1: Eliminar stub

```bash
rm tinywasm/svg/svg.go
```

Contiene solo `type Svg struct{}` — se reemplaza por los archivos del plan.

---

## Archivo: `svg/svg.go`

```go
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
```

---

## Archivo: `svg/sprite.go`

```go
package svg

import "github.com/tinywasm/fmt"

// Sprite holds SVG icon definitions for server-side sprite sheet injection.
// Build with New() and chain Add() calls.
// assetmin injects Sprite.String() inline at the top of <body>.
type Sprite struct {
	icons []iconEntry
}

type iconEntry struct {
	id      string
	content string // inner SVG content: <path fill="currentColor" d="..."/>
	viewBox string // e.g. "0 0 16 16"
}

// New creates an empty Sprite.
func New() *Sprite {
	return &Sprite{}
}

// Add registers an icon.
// id: referenced in Icon("id") and <use href="#id">
// content: inner SVG (NOT the wrapping <svg> tag) — must use fill="currentColor"
// viewBox: optional, defaults to "0 0 16 16"
func (s *Sprite) Add(id, content string, viewBox ...string) *Sprite {
	vb := "0 0 16 16"
	if len(viewBox) > 0 && viewBox[0] != "" {
		vb = viewBox[0]
	}
	s.icons = append(s.icons, iconEntry{id: id, content: content, viewBox: vb})
	return s
}

// Merge adds all icons from other into s.
// Used by assetmin to accumulate icons from multiple components into one master sprite.
func (s *Sprite) Merge(other *Sprite) *Sprite {
	if other == nil {
		return s
	}
	s.icons = append(s.icons, other.icons...)
	return s
}

// String renders the sprite as inline SVG with all <symbol> elements.
// Called by assetmin to inject at the top of <body>.
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
```

---

## Archivo: `svg/providers.go`

```go
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
```

---

## Tests movidos a svg/tests/

`uc_selectsearch_test.go` fue movido desde `tinywasm/html/tests/` a `tinywasm/svg/tests/` porque usa `Svg()` y `Use()` — builders que pertenecen a `tinywasm/svg`.

### Archivos en `svg/tests/`

| Archivo | Estado | Descripción |
|---------|--------|-------------|
| `uc_common_test.go` | ✅ creado | Helpers: `SetupDOM`, `TriggerEvent`, `GetRef`, `MockEvent` |
| `uc_selectsearch_test.go` | ⚠️ no compila | Usa `Svg()`, `Use()` de `tinywasm/svg` + `Input`, `Label`, `Div`, `Span` de `tinywasm/html` |

### Para que compile necesita:

1. `svg/go.mod` con:
   ```
   module github.com/tinywasm/svg
   require github.com/tinywasm/dom v<version>
   ```
2. `svg/tests/go.mod` con:
   ```
   module github.com/tinywasm/svg/tests
   require (
       github.com/tinywasm/svg  v<version>
       github.com/tinywasm/html v<version>
       github.com/tinywasm/dom  v<version>
       github.com/tinywasm/fmt  v<version>
   )
   ```
3. Builders `Svg()` y `Use()` implementados en `svg/svg.go`
4. Builders `Input`, `Label`, `Div`, `Span` implementados en `tinywasm/html`

### Transformaciones aplicadas al mover

- `package html_test` → `package svg_test`
- Agregado: `. "github.com/tinywasm/svg"` (dot import — `Svg`, `Use`)
- Mantenido: `. "github.com/tinywasm/html"` (dot import — `Input`, `Label`, `Div`, `Span`)
- Mantenido: `"github.com/tinywasm/dom"` (lifecycle: `dom.Render`, `dom.Element`, `dom.Event`)
- Sin cambios en lógica de test

---

## Tests unitarios: `svg/svg_test.go`

```go
package svg_test

import (
    "strings"
    "testing"
    . "github.com/tinywasm/svg"
)

func TestIcon_Structure(t *testing.T) {
    got := Icon("home", "nav-icon").String()
    if !strings.Contains(got, `href='#home'`) { t.Error("expected href") }
    if !strings.Contains(got, `aria-hidden='true'`) { t.Error("expected aria-hidden") }
    if !strings.Contains(got, `class='nav-icon'`) { t.Error("expected class") }
}

func TestSprite_String(t *testing.T) {
    s := New().
        Add("home", `<path fill="currentColor" d="M1 1"/>`, "0 0 576 512").
        Add("info", `<path fill="currentColor" d="m7 11h2v2h-2z"/>`)

    out := s.String()
    if !strings.Contains(out, `id="home"`) { t.Error("expected home symbol") }
    if !strings.Contains(out, `viewBox="0 0 576 512"`) { t.Error("expected custom viewBox") }
    if !strings.Contains(out, `viewBox="0 0 16 16"`) { t.Error("expected default viewBox") }
}

func TestSprite_Merge(t *testing.T) {
    a := New().Add("icon-a", "<path/>")
    b := New().Add("icon-b", "<path/>")
    a.Merge(b)
    out := a.String()
    if !strings.Contains(out, `id="icon-a"`) { t.Error("expected icon-a") }
    if !strings.Contains(out, `id="icon-b"`) { t.Error("expected icon-b") }
}

func TestSprite_Nil_Safe(t *testing.T) {
    var s *Sprite
    if s.String() != "" { t.Fatal("nil Sprite.String() should be empty") }
    if s.Merge(New()) == nil { t.Fatal("nil Merge should not panic") }
}
```

---

## Verificación

```bash
cd tinywasm/svg
go build ./...
gotest
```

---

## Uso en componentes (patrón post-migración)

### `mycomponent/mycomponent.go`
```go
import (
    . "github.com/tinywasm/svg"   // Icon(), Svg(), Use()
    . "github.com/tinywasm/html"  // Div, Span, etc.
    "github.com/tinywasm/dom"
)

func (c *NavItem) Render() *dom.Element {
    return Div(
        Icon("nav-home", "nav-item-icon"),
        Span("Home"),
    ).Class("nav-item")
}
```

### `mycomponent/svg.go` (`//go:build !wasm`)
```go
//go:build !wasm
package mycomponent

import "github.com/tinywasm/svg"

func (c *NavItem) IconSvg() *svg.Sprite {
    return svg.New().
        Add("nav-home", `<path fill="currentColor" d="M..."/>`, "0 0 576 512")
}
```

Ver `tinywasm/docs/MASTER_PLAN.md` para el orden global de ejecución.
