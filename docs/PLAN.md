# PLAN: tinywasm/svg — API tipada de iconos para el sprite

## Repositorio
`github.com/tinywasm/svg` — path local: `tinywasm/svg/`

## Objetivo
Eliminar las inconsistencias de la declaración stringly-typed actual de iconos,
con una API tipada donde **la definición y la referencia comparten un único valor**,
el `viewBox` es obligatorio, y el `fill` es infalible. Un typo pasa a ser **error
de compilación**, no un icono en blanco en runtime.

Cada proyecto declara sus propios iconos en su `svg.go` — no hay paquete central
de iconos. Los paths son específicos de cada dominio/proyecto.

---

## Problema (estado actual)

Hoy los iconos se declaran con strings sueltos:

```go
// declaración — componente/svg.go
svg.New().
    Add("icon-home", `<path fill="currentColor" d="M280..."/>`, "0 0 576 512").
    Add("icon-info", `<path fill="currentColor" d="m7 11h2..."/>`)  // ⚠️ sin viewBox

// referencia — componente/Render() (otro archivo)
svg.Icon("icon-home", "pd-nav-icon")   // ⚠️ string repetido, sin vínculo con Add()
```

Tres inconsistencias, todas **silenciosas** (compilan, fallan en runtime):

1. **`viewBox` con default peligroso.** `Add` cae a `"0 0 16 16"` si se omite. Un
   path diseñado para `576×512` renderizado en una caja `16×16` se recorta/desalinea.
   *Este es el bug observado en `platformd` (iconos desalineados).*
2. **ID stringly-typed y duplicado.** `"icon-home"` se escribe en `Add()` y otra vez
   en `Icon()`. Un typo en cualquiera → `<use>` apunta a un símbolo inexistente →
   icono vacío, sin error de compilación.
3. **`fill` olvidado.** Si se omite `fill="currentColor"` en el content, el icono
   se renderiza negro/invisible y no responde al theming por token CSS.

---

## Decisión de color: `fill="currentColor"` se queda en el símbolo

`currentColor` **no es un color**: es una indirección al `color` CSS del elemento.
Es el mecanismo correcto para un sprite y **debe permanecer fijo en el `<symbol>`**,
NO venir de una variable de tinywasm/css horneada en el path.

**Justificación:** un `<symbol>` se reutiliza por muchos `<use>`. Si se hornea un
color (o `var(--…)`) en el `fill` del path, se fuerza ese color en *todos* los usos
y se pierde el recoloreado por contexto. El diseño de `level_3` exige el mismo icono
en color secundario en el nav y **blanco** cuando está activo/hover — eso solo
funciona con `currentColor` + color puesto por el CSS del consumidor:

```go
// CSS del consumidor — única fuente de color, themeable vía tokens tinywasm/css
Rule(clsNavIcon, Color(tokenColorSecondary)),                     // normal
Rule(Selector(".pd-nav-active svg"), Color(tokenColorPrimary)),   // activo → blanco
```

> La definición del icono es **agnóstica al color** (siempre `currentColor`); el
> theming se centraliza en tokens CSS en el sitio de uso. Así ningún icono puede
> quedar "no-themeable".

---

## API propuesta (tinywasm/svg)

```go
package svg

// node es markup interno del cuerpo de un símbolo — privado a propósito.
// Los callers nunca nombran el tipo; solo pasan svg.Path(...) o svg.Raw(...)
// directamente a Define. La superficie pública queda en lo mínimo necesario.
type node string

// Path renderiza <path fill="currentColor" d="<d>"/>. El fill queda fijo en
// currentColor (infalible: nadie puede olvidarlo → ningún icono negro/invisible).
// Cubre el caso común: uno o varios vector paths.
func Path(d string) node

// Raw es el escape hatch para markup que Path no expresa (groups, circle, mask,
// gradientes). El autor decide dónde usar fill="currentColor" si quiere theming.
func Raw(s string) node

// Icon es una definición de icono tipada y autocontenida: id + viewBox + cuerpo,
// declarada UNA sola vez en el svg.go del proyecto. El mismo valor construye el
// <symbol> del sprite Y renderiza la referencia <use>, por lo que nunca pueden
// divergir.
type Icon struct {
    id      string
    viewBox string
    body    string
}

// Define crea un icono reutilizable. viewBox es OBLIGATORIO (sin default que
// recorte paths). body son uno o más node (Path/Raw).
func Define(id, viewBox string, body ...node) Icon Icon

// ID devuelve el id del símbolo (para <use href="#id">).
func (i Icon) ID() string

// Render devuelve <svg aria-hidden focusable=false class=...><use href="#id"/></svg>.
// Es la forma canónica de referenciar el icono en Render().
func (i Icon) Render(classes ...string) *dom.Element

// NewSprite construye un *Sprite a partir de iconos tipados.
// Reemplaza New().Add(...) con strings.
func NewSprite(icons ...Icon) *Sprite
```

### Patrón de uso en cada proyecto

**Archivo `svg.go` — declaración** (`//go:build !wasm`, solo SSR):

```go
//go:build !wasm
package platformd

import "github.com/tinywasm/svg"

// Vars package-level: cada icono declarado UNA sola vez con id + viewBox + paths.
var (
    iconHome = svg.Define("home", "0 0 576 512", svg.Path("M280..."))
    iconDoc  = svg.Define("doc", "0 0 16 16",
        svg.Path("M7..."),   // multi-path: tres paths dentro del mismo símbolo
        svg.Path("M11..."),
        svg.Path("M3..."),
    )
    iconLogo = svg.Define("logo", "0 0 32 32",
        svg.Raw(`<circle cx="16" cy="16" r="16"/>`), // escape hatch para non-path
    )
)

// IconSvg registra los símbolos del sprite que assetmin inyecta en el <body>.
func (p *Platform) IconSvg() *svg.Sprite {
    return svg.NewSprite(iconHome, iconDoc, iconLogo)
}
```

**Archivo `platformd.go` — referencia** (sin build tag, compila en wasm y !wasm):

```go
// HOY (stringly-typed — dos strings para el mismo icono, sin relación entre sí):
link.Add(svg.Icon("icon-home", "pd-nav-icon"))

// PROPUESTO (tipado — una sola var, imposible divergir):
link.Add(iconHome.Render(string(clsNavIcon)))
```

`iconHome` es la var declarada en `svg.go`. `Render` emite el `<svg><use>` con el
id correcto. `clsNavIcon` es la var `Class` del componente (`clsNavIcon Class = "pd-nav-icon"`),
convertida a `string` porque `Render` acepta `...string` — svg no importa tinywasm/css
para evitar dependencia circular, pero el llamador nunca escribe el literal:
si `clsNavIcon` se renombra en el CSS, el compilador avisa acá también.

### Cómo cierra cada inconsistencia
| Problema actual | Cómo lo elimina la API tipada |
|---|---|
| `viewBox` olvidado → recorte | `Define` exige `viewBox` (parámetro requerido, no hay default) |
| ID duplicado/typo silencioso | id vive UNA vez en la var `iconHome`; `Render()` y `NewSprite()` derivan del MISMO valor → typo = error de compilación |
| `fill` olvidado → icono negro | `Path()` inyecta `fill="currentColor"` siempre; no hay nada que olvidar |

---

## Compatibilidad con assetmin (SSR)

`Sprite` mantiene su representación interna (`iconEntry{id, content, viewBox}`) y
sus métodos `String()`, `MarshalJSON`/`UnmarshalJSON` y `Merge()` **sin cambios** —
assetmin sigue inyectando `Sprite.String()` inline en `<body>` y serializando por
JSON durante la extracción SSR. `NewSprite` construye esa misma data con seguridad
en compilación.

`Path(d)` produce exactamente `<path fill="currentColor" d="...">` — el sprite
resultante es byte-equivalent al markup anterior.

---

## Migración (break change limpio)

1. Agregar `Node`, `Path`, `Raw`, `Icon`, `Define`, `(Icon).ID/Render`, `NewSprite`
   en `tinywasm/svg`.
2. **Eliminar** `New()`, `Sprite.Add(id, content, viewBox string)` y la función
   libre `Icon(id string, classes ...string)` — sin `Deprecated:`, sin código legacy.
   `NewSprite(icons ...Icon)` es el único constructor; su nombre dice exactamente qué crea.
3. Migrar `tinywasm/layout/platformd/svg.go` a vars tipadas + `svg.NewSprite(...)` y
   `platformd.go` Render a `iconX.Render(string(clsNavIcon))`.
4. Bump de versión minor en `tinywasm/svg` — los consumidores que usen la API antigua
   no compilarán; deben migrar al mismo tiempo.

---

## Tests

- `Path(d)` genera `<path fill="currentColor" d="...">` exacto.
- `Define(id, viewBox, Path(d)).ID()` retorna `id`.
- `Icon.Render("cls")` emite `<svg aria-hidden focusable=false class=cls><use href="#id"/></svg>`.
- `NewSprite(icons...).String()` es byte-compatible con el `New().Add(...)` equivalente
  (guard de no-regresión para assetmin).
- Typo safety es de compilación — no requiere test.

---

## Documentación

Actualizar `tinywasm/svg/README.md`:
- Eliminar ejemplos con `New().Add(...)` y `svg.Icon(string,...)`
- Mostrar el patrón completo: `Define` + `Path`/`Raw` + `NewSprite` + `Render`
- Incluir el ejemplo de dos archivos (`svg.go` con build tag `!wasm` + referencia en `Render()`)
- Sección de color: por qué `currentColor` vive en el símbolo y el theming va en CSS

---

## Verificación

```bash
cd tinywasm/svg && go build ./... && gotest
```

Visual (con el daemon de app sobre `platformd`, hot reload ya operativo):
1. Los iconos del nav renderizan con su `viewBox` correcto (sin recorte).
2. El icono activo cambia a blanco (theming por `color:` token), confirmando
   que `currentColor` funciona.
3. Editar un `svg.Define` en `svg.go` refleja en caliente.
