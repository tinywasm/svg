# Agent Guide — `tinywasm/svg`

Constraints for agents modifying this library. Read before any change.

---

## Construction Harness — typed & explicit (the TinyWasm approach)

This library is part of TinyWasm's **construction harness**: the typed,
explicit API is what keeps an agent that doesn't know the library from building
wrong code. (Ecosystem rationale: `tinywasm/docs/ARNES_DE_CONSTRUCCION.md` and
`app-releases/docs/CONSTRUCTION_HARNESS.md`.)

**This library never uses `//go:build`.** It has two unconditional consumer
classes — the WASM browser client, and backend-only programs (`tinywasm/ssr`'s
extractor, `assetmin`) that need sprite construction at all times. A tag
inside the library cannot serve both correctly (the extractor would need to
build the library for `!wasm` specifically, which has nothing to do with its
own target). **The split is by Go package, not by build tag** — the consumer
decides what to import, and tags its OWN files.

## Package layout

| Package | Contains | Who imports it |
|---|---|---|
| `github.com/tinywasm/svg` | `type Icon string` + `Render()`. Imports only `tinywasm/dom`. | Everyone — reachable from WASM |
| `github.com/tinywasm/svg/sprite` | `Define`, `Sprite`, `Path`, `Raw`, `NewSprite`, `AddRaw`, `AddFile`, JSON (de)serialization. Imports `svg` + `github.com/tinywasm/json` + `github.com/tinywasm/model`. | Backend-only: a consumer's tagged `svg.go`, `tinywasm/ssr`, `assetmin` |

Never move `sprite`'s declarations into the root `svg` package, and never make
the root package depend on `sprite` (dependency points one way: `sprite` →
`svg`).

## The two halves of an icon (consumer side)

- **Reference (shared, any target):** `const iconX = svg.Icon("comp-x")` in
  untagged code, rendered with `iconX.Render(class)`. Only that string reaches
  the WASM binary.
- **Definition (consumer's own `!wasm` file):**
  `sprite.Define(iconX, "0 0 16 16", sprite.Path("..."))` inside the
  consumer's `//go:build !wasm` `svg.go`, exposed via
  `IconSvg() *sprite.Sprite`. `tinywasm/ssr` extracts it; `assetmin` merges
  all sprites and injects them inline at the top of `<body>` — there is no
  `/assets/icons.svg` URL; `href="#id"` always resolves without a network request.

`Icon.Render` is the ONLY public path to `<svg><use>`. Do not reintroduce
`Svg()`/`Use()` builders here or anywhere else.

## What this library CANNOT guarantee (know the limit)

Because `svg/sprite` is a plain importable Go package (no build tag gate),
NOTHING stops a consumer from importing it into a file reachable by a WASM
build — that mistake compiles successfully and silently grows the binary. This
library cannot prevent that by itself; it is a consumer-side responsibility.
Every consumer AGENTS.md MUST record this mandatory pre-publish check:

```bash
GOOS=js GOARCH=wasm go list -deps ./... | grep tinywasm/svg/sprite   # must be empty
```

When reviewing a consumer's `svg.go`, verify it carries `//go:build !wasm`
before it imports `svg/sprite` — that is the consumer's job, not this
library's, but it is the number one mistake this design is vulnerable to.

## Hard rules

- Root `svg` package (`icon.go`) imports ONLY `github.com/tinywasm/dom`. No
  stdlib, no `sprite`.
- `svg/sprite` serializes via `github.com/tinywasm/json` +
  `github.com/tinywasm/model` (`Encodable`/`Decodable`, plus `FielderSlice`
  for the bare-array wire shape `Sprite` needs). **Never** import stdlib
  `encoding/json` here — `tinywasm/json` is faster (no reflection, reused
  buffers) and it's the codec the ecosystem's own tests actually exercise, so
  using anything else means this package's serialization bugs go undetected
  by every other test that proves the codec correct. `Sprite` keeps
  `MarshalJSON`/`UnmarshalJSON` method NAMES (stdlib-compatible signatures)
  only because `tinywasm/ssr`'s outer envelope decode
  (legitimately `encoding/json`, backend tooling) needs to find them by
  interface — their BODIES delegate entirely to `tinywasm/json.Encode`/`Decode`.
- No `map`, no generics, no `any`. Value semantics; unexported struct fields.
- `Path(d)` hardcodes `fill="currentColor"` — icons are color-agnostic; theming
  happens in CSS at the use-site. Never emit fixed colors.
- `viewBox` is mandatory everywhere (`Define`, `AddRaw`, `AddFile`) — no implicit
  default. A symbol rendered in a box it was not drawn for is clipped or
  misaligned, and no default can recover the source coordinate system.
- `AddRaw` (body + explicit viewBox) and `AddFile` (a whole `.svg` file, viewBox
  parsed from its root element) exist for assetmin's raw file/favicon path only;
  they are not the consumer API. Parsing SVG markup belongs HERE, never in
  assetmin — assetmin bundles assets, it does not know the SVG format.
- No `regexp` and no stdlib in the file parser (`file.go`): plain byte scanning.

## Testing / publishing

- `gotest`, never `go test`. Stdlib assertions only (no testify).
- Publish with `gopush 'message'` (local developer only), never raw `git push`.
