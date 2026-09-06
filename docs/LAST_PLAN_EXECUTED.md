---
PLAN: "feat: sanear un SVG subido por un tercero antes de publicarlo"
TAG: v0.3.0
EXECUTOR: jules
REVIEWER: none
---

> Plan autocontenido: todo lo necesario para ejecutarlo está aquí.
> Se despacha con el flujo CodeJob. Ver skill: agents-workflow.
> Reglas del repo: [`AGENTS.md`](../AGENTS.md) en la raíz — léelo antes de tocar
> nada (las dos capas, la dirección de la dependencia, y qué puede alcanzar un
> binario WASM).

# Plan — `svg/sanitize`: un SVG de un desconocido no es un dibujo, es código

**Requisito previo**, porque este entorno no lo trae instalado:

```bash
go install webtyp.com/devflow/cmd/gotest@latest
```

## 1. El problema, con el caso real que lo destapó

`veltylabs/misitio` es un panel donde clientes de Velty suben el logo de su
sitio. Acepta `image/svg+xml`, guarda el archivo, y CI lo publica **tal cual** en
el dominio del cliente.

Un SVG no es un formato de imagen inerte: es XML con `<script>`, con
`<foreignObject>` —que abre HTML dentro—, con manejadores `onload`, y con
referencias externas. Servido desde el dominio del cliente, ejecuta en ese
origen. Es XSS con un formulario de subida por delante.

Este repo es el dueño de la gramática SVG del ecosistema, así que la limpieza va
aquí. Lo que **no** va aquí es dentro de `sprite`: ese paquete ingiere iconos
**de confianza**, escritos por desarrolladores en el propio repo, y los recorre
con búsqueda de strings (`viewBoxOf`, `innerOf` en `sprite/file.go`). Un saneador
necesita recorrer el árbol con una lista blanca. **Mezclar los dos caminos es
exactamente cómo termina colándose un `<script>`**: alguien reutiliza el lector
rápido de `sprite` para el archivo de un desconocido.

## 2. La API — paquete nuevo `sanitize`

Archivo nuevo: `sanitize/sanitize.go`, **con `//go:build !wasm` en la primera
línea**.

Esa etiqueta es deliberada y más estricta que la de `sprite`. El propio
[`AGENTS.md`](../AGENTS.md) admite que un paquete sin etiqueta no puede impedir
que alguien lo importe desde código WASM y engorde el binario en silencio. Con
`encoding/xml` dentro, ese accidente cuesta cientos de KB en un binario TinyGo
con un límite duro de 1 MB. Aquí sí se puede impedir, así que se impide.

```go
package sanitize

// Clean devuelve el SVG sin nada ejecutable ni nada que salga a la red.
//
// Elimina lo que no entiende y falla ante lo que sólo puede ser un ataque.
func Clean(src []byte) ([]byte, error)
```

### 2.1 — Las dos políticas, y por qué son distintas

**Rechazar** —devolver error y no publicar nada— cuando el archivo contiene:

| Encontrado | Por qué no se limpia y ya |
|---|---|
| `<script>` | un logo no lleva scripts: o es un ataque o es un archivo equivocado, y publicar una versión desactivada esconde las dos cosas |
| un atributo `on*` (`onload`, `onclick`, …) | lo mismo |
| `<foreignObject>` | mete HTML arbitrario dentro del SVG; nada legítimo en un logo lo necesita |

Errores textuales, con `webtyp.com/fmt`:

```
svg: el archivo contiene <script>: un logo no ejecuta codigo
svg: el archivo contiene el manejador onload: un logo no ejecuta codigo
svg: el archivo contiene <foreignObject>: un logo no incrusta HTML
```

**Eliminar y continuar** para todo lo demás que no esté en la lista blanca: un
SVG exportado desde Illustrator o Figma trae metadatos, `<style>`, comentarios y
atributos de editor, y rechazarlo por eso sería rechazar a la mitad de los
clientes por algo que no hace daño una vez quitado.

### 2.2 — La lista blanca

Sólo estos elementos sobreviven:

```
svg  g  defs  symbol  use  title  desc
path  rect  circle  ellipse  line  polyline  polygon
linearGradient  radialGradient  stop  clipPath  mask
text  tspan
```

Sólo estos atributos sobreviven:

```
xmlns  xmlns:xlink  viewBox  width  height  preserveAspectRatio
d  points  x  y  x1  y1  x2  y2  cx  cy  r  rx  ry
fill  fill-opacity  fill-rule  stroke  stroke-width  stroke-linecap
stroke-linejoin  stroke-dasharray  stroke-opacity  opacity
transform  gradientUnits  gradientTransform  offset  stop-color
stop-opacity  clip-path  mask  class  id  role  aria-label
```

Reglas adicionales, porque un atributo de la lista puede seguir apuntando afuera:

- `href` y `xlink:href` se conservan **sólo** si su valor empieza con `#`. Una
  referencia local resuelve dentro del archivo; cualquier otra cosa —`http://`,
  `data:`, `javascript:`— sale a la red o ejecuta, y se elimina.
- `<style>` se elimina siempre: admite `@import` y `url(...)`, que salen a la
  red, y un logo no lo necesita.
- Cualquier valor de atributo que, sin espacios y en minúsculas, empiece con
  `javascript:` elimina el atributo entero.

### 2.3 — Qué devuelve

- El SVG serializado de nuevo, con un único elemento raíz `<svg>`.
- Si el XML no es válido o no hay raíz `<svg>`: error. No intentes reparar:
  `svg: el archivo no es un SVG valido`.
- **`Clean` es idempotente**: limpiar lo ya limpio devuelve exactamente lo
  mismo. Es lo que permite ejecutarlo dos veces —al subir y al publicar— sin
  pensarlo.

Usa `encoding/xml` de la stdlib para recorrer y reescribir. Es host, es
`!wasm`, y aquí la stdlib es correcta y esperada.

## 3. Tests — `tests/sanitize_test.go`

`gotest`, nunca `go test`. Ojo: `tests/` de este repo es **un módulo aparte**,
con su propio `go.mod` y un `replace` al padre; el archivo nuevo va ahí, y como
el saneador es `!wasm`, el test lleva `//go:build !wasm` (el que ya existe es
`wasm`, no lo toques).

| Test | Qué fija |
|---|---|
| `TestCleanRejectsScript` | `<script>alert(1)</script>` → error, y el mensaje nombra `<script>`. |
| `TestCleanRejectsEventHandler` | `<svg onload="...">` → error nombrando `onload`. |
| `TestCleanRejectsForeignObject` | `<foreignObject>` → error. |
| `TestCleanStripsStyle` | Un `<style>@import url(...)</style>` desaparece y el resto del dibujo sobrevive. |
| `TestCleanStripsExternalHref` | `<use href="https://evil/x">` pierde el atributo; `<use href="#logo">` lo conserva. |
| `TestCleanStripsUnknownAttributes` | Un atributo de editor (`inkscape:label`) desaparece; `d` y `fill` sobreviven. |
| `TestCleanKeepsRealLogo` | Un SVG con `viewBox`, dos `<path>` y un `<linearGradient>` sale con los tres elementos y sigue siendo XML válido. |
| `TestCleanIsIdempotent` | `Clean(Clean(x)) == Clean(x)`, byte a byte. |
| `TestCleanRejectsGarbage` | Bytes que no son XML, y un XML válido que no es SVG → error. |

## 4. Documentación

- [`README.md`](../README.md): una tercera fila en la tabla de paquetes —
  `svg/sanitize`, `!wasm`, "limpia SVG de terceros"— dejando explícito que
  `sprite` es para iconos **de confianza** y `sanitize` para archivos subidos.
- [`AGENTS.md`](../AGENTS.md): la misma distinción, en la sección de las dos
  capas. Es la regla que evita que alguien reutilice `sprite` para leer lo que
  subió un desconocido.

Ningún documento debe citar `docs/PLAN.md`: este archivo se borra al publicar.

## 5. Criterios de aceptación

- [ ] `gotest` en verde.
- [ ] `gofmt -l .` vacío.
- [ ] `head -1 sanitize/*.go` → todos empiezan con `//go:build !wasm`.
- [ ] `grep -rn "encoding/xml" --include=*.go . | grep -v sanitize/ | grep -v tests/` → vacío: el parser no se filtra al resto del repo.
- [ ] `grep -rn "sanitize" sprite/ icon.go` → vacío: los dos caminos no se tocan.
- [ ] `go.mod` **sin dependencias nuevas**: todo es stdlib más `webtyp/fmt`.
- [ ] Los nueve tests de §3 existen y pasan.

## 6. Anti-footguns

1. **No toques `sprite` ni el paquete raíz.** La dirección de la dependencia es
   `sprite → svg` y no cambia; `sanitize` no depende de ninguno de los dos.
2. **No conviertas esto en un validador de dibujos.** No comprueba que el SVG se
   vea bien, ni que tenga `viewBox`, ni que sea cuadrado. Sólo quita lo que
   ejecuta o sale a la red. Las medidas del logo las valida
   `webtyp.com/image/favicon`.
3. **No uses una lista negra.** Enumerar lo prohibido deja fuera lo que se
   invente mañana; la lista blanca de §2.2 deja fuera todo lo que no esté
   escrito. Si un logo legítimo pierde algo, se agrega a la lista con su caso de
   prueba.
4. `docs/PLAN.md` (este archivo) no se renombra ni se borra, y su frontmatter
   —`PLAN`, `TAG`, `EXECUTOR`, `STATUS`, `SESSION`, `PR`— **no se edita a mano**.
