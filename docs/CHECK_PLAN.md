# PLAN: tinywasm/svg — JSON round-trip del Sprite (para el IPC del codegen ssr)

## Repositorio
`github.com/tinywasm/svg` — path local: `tinywasm/svg/` (publicado v0.0.2)

## Contexto
El extractor SSR (`tinywasm/ssr`) corre el codegen en **otro proceso** (`go run`) y devuelve los
assets por **JSON**. `IconSvg()` ahora retorna `*svg.Sprite`, así que el sprite debe cruzar el
límite de proceso. Hoy `Sprite` tiene solo campos privados (`icons []iconEntry`) → el JSON por
defecto sería `{}` y se perderían los íconos.

Ver `tinywasm/ssr/docs/PLAN.md` Paso 3 y `tinywasm/assetmin/docs/PLAN.md` Cambio 5d/7.

---

## Cambio: `MarshalJSON` / `UnmarshalJSON` en `Sprite`

En `svg/sprite.go`, agregar round-trip JSON sobre una forma exportada de los íconos:

```go
import "encoding/json"

// jsonIcon es la forma serializable de un icono (campos exportados).
type jsonIcon struct {
    ID      string `json:"id"`
    Content string `json:"content"`
    ViewBox string `json:"viewBox"`
}

func (s *Sprite) MarshalJSON() ([]byte, error) {
    out := make([]jsonIcon, len(s.icons))
    for i, e := range s.icons {
        out[i] = jsonIcon{ID: e.id, Content: e.content, ViewBox: e.viewBox}
    }
    return json.Marshal(out)
}

func (s *Sprite) UnmarshalJSON(b []byte) error {
    var in []jsonIcon
    if err := json.Unmarshal(b, &in); err != nil {
        return err
    }
    s.icons = make([]iconEntry, len(in))
    for i, e := range in {
        s.icons[i] = iconEntry{id: e.ID, content: e.Content, viewBox: e.ViewBox}
    }
    return nil
}
```

> Serializa como array de íconos (no como objeto). Round-trip estable: `Marshal` → `Unmarshal`
> reconstruye el mismo `Sprite` (incluido el orden, que importa para la salida del sprite).

---

## Tests

`svg/sprite_json_test.go`:
- Round-trip: `New().Add("a", "<path/>").Add("b", "<rect/>", "0 0 24 24")` → `Marshal` →
  `Unmarshal` → `String()` idéntico al original.
- Sprite vacío y `nil` no entran en pánico.

## Verificación
```bash
cd tinywasm/svg
go build ./...
gotest
gopush   # nueva versión (v0.0.3): assetmin/ssr la requieren
```

Ver `tinywasm/docs/MASTER_PLAN.md` para el orden global.
