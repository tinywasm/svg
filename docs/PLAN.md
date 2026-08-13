---
PLAN: "fix: sprite.Merge no deduplica ni respeta al receptor; String() pierde el envoltorio"
EXECUTOR: jules
REVIEWER: none
STATUS: running
SESSION: 2188326099226198193
---

> Este plan se despacha con el flujo CodeJob. Ver skill: agents-workflow.
>
> Forma parte de un cambio con ruptura coordinado desde
> https://github.com/tinywasm/app/blob/main/docs/PLAN.md

# Plan — `svg/sprite` debe cerrar su propio contrato

## El problema

Tres cosas que son función pura de un `Sprite` se resuelven hoy **fuera** de este
paquete, en `github.com/tinywasm/assetmin`, con una semántica distinta a la de
aquí. El resultado son dos políticas de deduplicación conviviendo y un bug de
aliasing con test de regresión en otro repo.

### Defecto 1 — `Merge` no deduplica

```go
// sprite.go:67
func (s *Sprite) Merge(other *Sprite) *Sprite {
	s.icons = append(s.icons, other.icons...)
	return s
}
```

Concatena. Dos módulos que declaran el mismo icono producen dos `<symbol>` con
el mismo `id`, lo que es HTML inválido.

`assetmin` compensa aguas abajo con su propia deduplicación —primero gana,
ordenado por nombre de módulo— que este paquete desconoce. Por eso hay dos
semánticas: `sitec` fusiona paquetes dentro de un módulo con `Merge` (sin
deduplicar) y `assetmin` fusiona entre módulos (deduplicando). La política debe
vivir aquí, una sola vez.

### Defecto 2 — `Merge` aliasa y muta al receptor

```go
if s == nil {
	return other      // <-- devuelve el MISMO puntero
}
```

Cuando el receptor es nil devuelve `other` sin copiar. El llamador cree tener un
sprite nuevo y en realidad tiene el del primer paquete; el siguiente `Merge`
le añade en el sitio y corrompe permanentemente el sprite cacheado.

Está documentado y con test de regresión **en otro repo**
(`tinywasm/sitec/tests/mergeicons_test.go`), que describe el efecto observado:

> *icons duplicate on some rebuilds and, once assetmin dedups by id, the net
> served set no longer matches any single module — the crudview/targetlist
> symbols stop appearing while platformd's remain.*

Un invariante de este paquete no puede estar defendido solo por el test de un
consumidor.

### Defecto 3 — `String()` pierde el envoltorio cuando no hay iconos

```go
if s == nil || len(s.icons) == 0 {
	return ""
}
```

`assetmin` parchea el vacío **después**, reinyectando el envoltorio a mano
`<svg aria-hidden="true" style="display:none"></svg>` con el comentario de que
hace falta un punto de inyección estable. Ese envoltorio es markup: lo emite
`String()` o no lo emite nadie.

### Defecto 4 — no se puede preguntar si un ID ya existe

`assetmin.checkIconID` renderiza el sprite **entero a string** y busca
`id="…"` por substring para detectar duplicados. Un sprite sabe qué IDs
contiene; consultarlo no debería requerir parsear su propio markup.

---

## Etapa 1 — `Merge` copia y deduplica

```go
// Merge devuelve un sprite NUEVO con los iconos de s seguidos de los de other,
// conservando la primera aparición de cada ID. Nunca muta a s ni a other, y
// nunca devuelve un puntero recibido: el llamador siempre obtiene un sprite
// propio.
func (s *Sprite) Merge(other *Sprite) *Sprite
```

Dos cambios de comportamiento, ambos con ruptura y deliberados:

1. **No muta.** Ni al receptor ni al argumento. Con `s == nil` devuelve una
   copia de `other`, no `other`.
2. **Deduplica por `Definition.Icon.ID()`**, conservando la primera aparición.

El orden de los iconos supervivientes es el de inserción, que es determinista
si el llamador fusiona en orden determinista.

**Aceptación:**

- Un test que fusiona dos sprites con un ID común y afirma que el resultado
  tiene ese ID una sola vez, con el cuerpo del primero.
- Un test que afirma que `a.Merge(b)` no cambia `a.Len()` ni `b.Len()`.
- Un test que afirma que `(*Sprite)(nil).Merge(b)` devuelve un puntero
  **distinto** de `b`, y que mutar el resultado no altera `b`.

## Etapa 2 — `MergeAll` para el caso de N sprites

```go
// MergeAll fusiona sprites en orden, conservando la primera aparición de cada
// ID. Es el único punto donde vive la política de deduplicación del ecosistema:
// los consumidores fusionan por paquete y por módulo llamando aquí, no
// reimplementándola.
func MergeAll(sprites ...*Sprite) *Sprite
```

Ignora los `nil`. Con cero sprites, o con todos vacíos, devuelve un sprite vacío
no-nil (ver etapa 3).

Esta es la función que sustituye la deduplicación manual que hoy vive en
`assetmin/svg.go`. El **orden** lo decide el llamador —`sitec` ordena por nombre
de módulo para que el resultado sea estable entre escaneos—; la **política** vive
aquí.

**Aceptación:** un test con tres sprites donde el segundo y el tercero repiten
un ID del primero afirma que solo sobrevive el del primero, y que el orden de
los demás se conserva.

## Etapa 3 — `String()` siempre emite el envoltorio

```go
// String renderiza el sprite como SVG inline. Un sprite sin iconos emite el
// envoltorio vacío, no una cadena vacía: el documento que lo inyecta necesita un
// punto de inserción estable, y componerlo fuera obliga a cada consumidor a
// duplicar el markup.
func (s *Sprite) String() string
```

Con cero iconos devolver exactamente:

```html
<svg aria-hidden="true" style="display:none"></svg>
```

Un `*Sprite` nil sigue devolviendo `""` — no hay sprite del que hablar. La
distinción "nil = no hay sprite" contra "vacío = hay sprite sin iconos" es la que
permite al consumidor decidir sin adivinar.

Declarar el envoltorio como constante del paquete; hoy la cadena está escrita a
mano en dos repos.

**Aceptación:** `NewSprite().String()` devuelve el envoltorio vacío;
`(*Sprite)(nil).String()` devuelve `""`.

## Etapa 4 — consultar IDs sin renderizar

```go
// Has reporta si el sprite ya contiene un icono con ese ID.
func (s *Sprite) Has(id string) bool

// IDs devuelve los IDs contenidos, en orden de inserción.
func (s *Sprite) IDs() []string
```

Sustituyen el `strings.Contains(render, "id=\""+id+"\"")` que hoy vive en
`assetmin`. Un sprite nil responde `false` y `nil`.

**Aceptación:** un test afirma que `Has` encuentra un ID cuyo cuerpo contiene la
subcadena `id="otro"`, es decir que no se resuelve por texto renderizado.

---

## Restricciones

### Este paquete SÍ se compila a WASM

`github.com/tinywasm/svg` es código de frontend: se compila dentro del binario
del navegador. **Aplica la regla del ecosistema**: nada de biblioteca estándar.
Usar `github.com/tinywasm/fmt` en vez de `fmt`/`strings`/`strconv`, como ya hace
`sprite.go` con `fmt.Builder`.

Este es el caso contrario al de `sitec`, `assetmin` y `app`, que son herramienta
de backend y usan stdlib legítimamente. No traslades aquí lo que valga allí.

### Sin strings hardcodeados

El envoltorio vacío es una constante del paquete, no un literal repetido.

### Cambio con ruptura

`Merge` cambia de semántica: deja de mutar y empieza a deduplicar. No dejes la
versión antigua bajo otro nombre ni un flag para elegir. Los consumidores
(`sitec`, `assetmin`) se actualizan en sus propios planes.

### No hacer

- No añadir llamadas a `gopush` ni a `codejob`.
- No crear carpetas `internal/`.

## Etapas

| # | Alcance | Archivos | Aceptación |
|---|---|---|---|
| 1 | `Merge` copia y deduplica | `sprite/sprite.go` | no muta receptor ni argumento; ID repetido sobrevive una vez |
| 2 | `MergeAll(...)` | `sprite/sprite.go` | política de dedupe en un solo lugar |
| 3 | `String()` emite el envoltorio vacío | `sprite/sprite.go` | `NewSprite().String()` no es `""` |
| 4 | `Has` / `IDs` | `sprite/sprite.go` | sin búsqueda por substring en el markup |

Puerta final:

```
go test ./...
```
