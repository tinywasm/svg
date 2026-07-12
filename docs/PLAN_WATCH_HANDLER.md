# PLAN — `tinywasm/svg`: `svg/watch` — devwatch handler that catches the leak at save-time

> This plan is dispatched via the CodeJob workflow. See skill: agents-workflow.
> Master plan: https://github.com/tinywasm/tinywasm/blob/main/docs/SVG_ICON_HARNESS_MASTER_PLAN.md
> Queued after `docs/PLAN.md` in this repo (the `svg`/`svg/sprite` package
> split) — does not block it, but needs `github.com/tinywasm/svg/sprite` to
> exist as an import path to detect, so run this SECOND.
>
> Repo rules: `AGENTS.md` at this repo's root.

## Context (zero-context summary)

`tinywasm/svg` is split into two packages: `svg` (just `type Icon string` +
`Render()`, safe for WASM) and `svg/sprite` (geometry/JSON, backend-only by
convention — it has no build tag of its own, so nothing stops a `.go` file
from importing it without `//go:build !wasm`, which silently ships path data
into the browser bundle). The library's README documents a manual pre-publish
check:

```bash
GOOS=js GOARCH=wasm go list -deps ./... | grep tinywasm/svg/sprite
```

Nobody runs manual checks by habit. `tinywasm/app`'s hot-reload already has a
generic extension point for exactly this class of problem:
`devwatch.FilesEventHandlers` (interface in
`github.com/tinywasm/devwatch/devwatch.go`):

```go
type FilesEventHandlers interface {
	NewFileEvent(fileName, extension, filePath, event string) error
	SupportedExtensions() []string
	UnobservedFiles() []string
	MainInputFileRelativePath() string
}
```

`tinywasm/ormc` already implements this to regenerate `*_orm.go` on every
`model.go`/`models.go` save (`ormc/watch.go`). This plan gives `svg` an
equivalent handler, scoped to `svg.go` files, that detects the leak and fails
loudly (does NOT auto-fix — a caught import in shared code might be a genuine
design mistake, not a forgotten tag; silent correction would mask that).

**Decision already made, do not revisit:** no centralized AST/parse bus in
`devwatch`/`app`. Each handler (`ormc`, `ssr/scanner`, `depfind`, this one)
parses independently; `devwatch` only dispatches `(fileName, extension,
filePath, event string)`. This handler reuses the mtime-cache parsing pattern
already proven in `ssr/scanner.go` (parse only on mtime change, cache the
result) rather than inventing shared infrastructure — evaluated and rejected
as overkill for a per-file, imports-only parse.

## Target package layout

```
svg/
└── watch/
    ├── guard.go        # LeakGuard type + devwatch.FilesEventHandlers methods
    └── guard_test.go
```

## Target API

```go
package watch

// LeakGuard implements devwatch.FilesEventHandlers. It watches svg.go files
// and fails loudly if one imports github.com/tinywasm/svg/sprite without a
// leading //go:build !wasm constraint — the mistake that silently ships
// sprite geometry + encoding/json into the WASM binary.
type LeakGuard struct {
	// unexported: mtime cache mirroring ssr/scanner.go's fileImportCache
}

// New constructs a ready-to-register LeakGuard.
func New() *LeakGuard

func (g *LeakGuard) NewFileEvent(fileName, extension, filePath, event string) error
func (g *LeakGuard) SupportedExtensions() []string  // []string{".go"}
func (g *LeakGuard) UnobservedFiles() []string       // nil
func (g *LeakGuard) MainInputFileRelativePath() string // ""
```

## Stages

### Stage 1 — `guard.go`: scope and cache

Mirror `ssr/scanner.go`'s cache shape exactly (same field names/semantics
where sensible, so a future reader recognizes the pattern):

```go
package watch

import (
	"go/parser"
	"go/token"
	"os"
	"strings"
	"sync"
	"time"
)

const spriteImportPath = "github.com/tinywasm/svg/sprite"
const watchedFileName = "svg.go"

type fileCheckCache struct {
	mtime time.Time
	err   error
}

type LeakGuard struct {
	mu    sync.RWMutex
	cache map[string]fileCheckCache
}

func New() *LeakGuard {
	return &LeakGuard{cache: make(map[string]fileCheckCache)}
}

func (g *LeakGuard) SupportedExtensions() []string    { return []string{".go"} }
func (g *LeakGuard) UnobservedFiles() []string          { return nil }
func (g *LeakGuard) MainInputFileRelativePath() string  { return "" }
```

### Stage 2 — `NewFileEvent`: scope to `svg.go`, mtime-cache, check

```go
func (g *LeakGuard) NewFileEvent(fileName, extension, filePath, event string) error {
	if fileName != watchedFileName {
		return nil // scope: only svg.go, mirrors ormc's model.go/models.go filter
	}

	info, err := os.Stat(filePath)
	if err != nil {
		return nil // file removed/renamed mid-event; nothing to check
	}

	g.mu.RLock()
	cached, ok := g.cache[filePath]
	g.mu.RUnlock()
	if ok && cached.mtime.Equal(info.ModTime()) {
		return cached.err
	}

	checkErr := checkFile(filePath)

	g.mu.Lock()
	g.cache[filePath] = fileCheckCache{mtime: info.ModTime(), err: checkErr}
	g.mu.Unlock()

	return checkErr
}
```

### Stage 3 — `checkFile`: import + build-tag detection

```go
func checkFile(filePath string) error {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filePath, nil, parser.ImportsOnly|parser.ParseComments)
	if err != nil {
		return nil // let the compiler report syntax errors; not this handler's job
	}

	importsSprite := false
	for _, imp := range f.Imports {
		if strings.Trim(imp.Path.Value, `"`) == spriteImportPath {
			importsSprite = true
			break
		}
	}
	if !importsSprite {
		return nil
	}

	if hasWasmExclusionTag(f) {
		return nil
	}

	return fmt.Errorf(
		"%s imports %q without a leading \"//go:build !wasm\" constraint — "+
			"this ships sprite geometry and encoding/json into the WASM binary. "+
			"Add \"//go:build !wasm\" as the file's first line.",
		filePath, spriteImportPath,
	)
}
```

Use `github.com/tinywasm/fmt` for the error (`fmt.Err(...)`), NOT stdlib
`fmt` — `svg/watch` is a dev-tooling package but this repo's own convention
(per `AGENTS.md`) is `tinywasm/fmt` everywhere except inside `svg/sprite`
(which has its own documented stdlib exception). Verify the exact `Err`
signature in `tinywasm/fmt` before writing this — match its existing call
style used elsewhere in this repo (`sprite.go`'s `fmt.Err` calls, pre-split).

`hasWasmExclusionTag(f)` inspects `f.Comments` for a comment group entirely
before `f.Package` (the package keyword's position) containing a line
matching `//go:build` (or legacy `// +build`) whose expression contains
`!wasm`. Implement with a simple substring/regex check — this ecosystem
already uses regex-based textual tag detection elsewhere (`ssr/invoke.go`'s
`reIconSvg` family) as precedent; do not implement full build-constraint
boolean evaluation (overkill for this scope).

### Stage 4 — tests

```go
func TestLeakGuard_MissingTag(t *testing.T) {
	// svg.go WITHOUT //go:build !wasm, importing svg/sprite -> error
}
func TestLeakGuard_CorrectTag(t *testing.T) {
	// svg.go WITH //go:build !wasm, importing svg/sprite -> nil
}
func TestLeakGuard_NoSpriteImport(t *testing.T) {
	// svg.go without the import at all -> nil (nothing to guard)
}
func TestLeakGuard_IgnoresOtherFiles(t *testing.T) {
	// fileName != "svg.go" -> nil, no parse attempted
}
func TestLeakGuard_MtimeCache(t *testing.T) {
	// second NewFileEvent call with unchanged mtime returns cached result
	// without re-parsing (assert via a parse-count hook or file-removal trick)
}
```

Run with `gotest` (never `go test`). Stdlib assertions only.

### Stage 5 — README note

Add one paragraph to `svg/README.md`'s "The one thing this library cannot
enforce" section: this manual check is now automated during development via
`svg/watch.LeakGuard`, registered in `tinywasm/app`'s hot-reload (see
`app/docs/PLAN_SVG_LEAK_GUARD.md`) — the manual `go list -deps` command
remains documented as the CI/pre-publish fallback for environments without
the watcher running.

### Stage 6 — verification

```bash
gotest
grep -n "package watch" svg/watch/guard.go   # sanity
```

## Code quality checklist (mandatory)

- No hardcoded repeated strings — `spriteImportPath`, `watchedFileName` are
  named constants (already shown above); do not inline the literals a second
  time anywhere in this package.
- Thin surface: `LeakGuard`, `New` exported; `fileCheckCache`,
  `checkFile`, `hasWasmExclusionTag` stay unexported.
- No `map` beyond the mtime cache (justified: keyed by file path, same
  justification `ssr/scanner.go` already uses in this exact ecosystem).
- This package is backend/dev-tooling only by nature (it's a devwatch
  handler, never imported by a component) — no build tag needed on it, but
  do NOT import it from any component or shared code; only `tinywasm/app`
  imports it (Phase G).
- Never run `gopush` or `codejob`.

## Stages table

| # | Stage | Files | Done |
|---|---|---|---|
| 1 | Package scope + cache type | `watch/guard.go` | ☐ |
| 2 | `NewFileEvent` scoping + cache lookup | `watch/guard.go` | ☐ |
| 3 | `checkFile` import + tag detection | `watch/guard.go` | ☐ |
| 4 | Tests | `watch/guard_test.go` | ☐ |
| 5 | README note | `README.md` | ☐ |
| 6 | Verification | — | ☐ |
