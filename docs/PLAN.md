---
PLAN: "fix!: quitar Sprite.Merge — un resultado que se puede descartar en silencio"
TAG: v0.2.0
---

> Este plan se despacha con el flujo CodeJob. Ver skill: agents-workflow.
>
> Es la **etapa A de una ola de 4** y es un **gate**: `tinywasm/sitec` no puede
> empezar su etapa hasta que esto esté publicado. Orquestador:
> https://github.com/tinywasm/docs/blob/main/BUILD_PIPELINE_MASTER_PLAN.md

# Plan — quitar `Sprite.Merge`

## 1. Por qué

`sprite.Merge` es una **función pura expuesta como método**:

```go
// sprite/sprite.go
func (s *Sprite) Merge(other *Sprite) *Sprite {
	return MergeAll(s, other)
}
```

No muta el receptor: devuelve un sprite nuevo. Pero se *lee* como un método
mutador, y Go acepta `x.Merge(y)` como sentencia, así que descartar el
resultado compila sin una sola advertencia.

Eso ocurrió. `tinywasm/sitec` tenía exactamente esto en sus dos puntos de
mezcla:

```go
merged.Icons.Merge(out.Icons)   // resultado descartado
mergedSprite.Merge(sp)          // resultado descartado
```

Consecuencia: **todos los iconos del ecosistema desaparecieron**. El sprite se
emitía vacío y cada `<use href="#…">` del marcado apuntaba a un símbolo que no
existía. El sitio cargaba con 200 en todo y sin errores de JavaScript; los
iconos simplemente no se dibujaban. Nadie lo notó durante varias versiones.

El método no aporta nada que `MergeAll` no dé, y su forma invita al error. El
harness (`app-releases/docs/CONSTRUCTION_HARNESS.md`) lo cubre en dos puntos:
*"One way to do each thing"* y *"Silent failures … turn them into compile
errors"*. Quitándolo, `merged.Icons.Merge(out.Icons)` deja de compilar.

## 2. Contexto del repo para un agente sin contexto previo

- Módulo: `github.com/tinywasm/svg`. `docs/PLAN.md` va junto a `go.mod`.
- El paquete afectado es `sprite/`. Archivos: `sprite.go`, `merge.go`,
  `file.go`, más sus `_test.go`.
- `MergeAll(sprites ...*Sprite) *Sprite` ya existe en `sprite/merge.go`, ya es
  la política de deduplicación del ecosistema (conserva la primera aparición de
  cada id) y ya devuelve un `*Sprite` nuevo sin mutar ningún argumento. **No la
  toques.**
- **Este paquete compila a WASM**: nada de librería estándar. Usa
  `github.com/tinywasm/fmt`, nunca `errors`/`strconv`/`strings`.
- Prohibidas las cadenas repetidas en la lógica: todo literal repetido va a una
  constante con nombre.

## 3. Pasos

### Paso 1 — borrar el método

En `sprite/sprite.go`, eliminar por completo:

```go
func (s *Sprite) Merge(other *Sprite) *Sprite {
	return MergeAll(s, other)
}
```

Junto con el bloque de comentario que lo precede (el que explica la versión
antigua que aliasaba el sprite cacheado). Ese comentario documenta un fallo del
método que ya no existirá; su parte útil —por qué `MergeAll` devuelve un sprite
fresco y no muta argumentos— **ya está** en el comentario de `MergeAll` en
`merge.go`. No lo dupliques allí.

### Paso 2 — migrar los tests

`sprite/merge_test.go` usa el método en tres sitios:

- línea ~21: `got := a.Merge(b).IDs()` → `got := MergeAll(a, b).IDs()`
- línea ~40: `_ = a.Merge(b)` → `_ = MergeAll(a, b)`
- línea ~56: `got := (*Sprite)(nil).Merge(b)` → `got := MergeAll(nil, b)`

El tercero comprueba el caso del receptor nil. `MergeAll` ya ignora entradas
nil, así que la afirmación del test no cambia: sigue esperando el contenido de
`b`. Verifica que sigue pasando; si no, el defecto está en `MergeAll` y hay que
arreglarlo ahí, no reintroducir el método.

### Paso 3 — test de regresión de la forma de uso

Añade en `sprite/merge_test.go`:

```go
// TestMergeAllIsTheOnlyCombiner fija la razón de que Sprite.Merge no exista.
// Era una función pura con forma de método: x.Merge(y) compila como sentencia
// y tira el resultado sin aviso. Esa exacta llamada, en tinywasm/sitec, dejó
// el sprite vacío y con él todos los iconos del ecosistema.
func TestMergeAllIsTheOnlyCombiner(t *testing.T) {
	a := NewSprite(Define("a", "0 0 1 1", Path("M0 0")))
	b := NewSprite(Define("b", "0 0 1 1", Path("M1 1")))

	got := MergeAll(a, b)

	if got.Len() != 2 {
		t.Fatalf("MergeAll debe combinar ambos, obtuve %d", got.Len())
	}
	if a.Len() != 1 || b.Len() != 1 {
		t.Errorf("MergeAll no debe mutar sus argumentos: a=%d b=%d", a.Len(), b.Len())
	}
}
```

## 4. Criterios de aceptación

Cada uno verificable con un comando:

1. `grep -rn "func (s \*Sprite) Merge" .` → **vacío**.
2. `grep -rn "\.Merge(" .` → **vacío** (ninguna llamada al método queda, ni en
   código ni en tests).
3. `go build ./...` y `go test ./...` en verde.
4. `go vet ./...` sin salida.

## 5. Qué NO hacer

- **No** cambies `MergeAll`: su firma, su política de deduplicación y su
  garantía de no mutar argumentos son las correctas y otras librerías dependen
  de ellas.
- **No** añadas un método sustituto con otro nombre (`Combine`, `Plus`, …). El
  objetivo es que exista **una** forma, no otra forma.
- **No** toques `tinywasm/sitec` desde aquí. Sus dos llamadas ya asignan el
  resultado (`merged.Icons = merged.Icons.Merge(...)`), así que hoy funcionan;
  migrarlas a `MergeAll` es la etapa B, en su propio repositorio.
- **No** introduzcas librería estándar: este paquete viaja dentro del binario
  WASM.

## 6. Etapas

| # | Archivo | Qué |
|---|---|---|
| 1 | `sprite/sprite.go` | Borrar `func (s *Sprite) Merge` y su comentario |
| 2 | `sprite/merge_test.go` | Migrar las 3 llamadas a `MergeAll` |
| 3 | `sprite/merge_test.go` | Añadir `TestMergeAllIsTheOnlyCombiner` |
