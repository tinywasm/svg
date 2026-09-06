package watch

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"sync"
	"time"

	"webtyp.com/fmt"
)

const spriteImportPath = "webtyp.com/svg/sprite"
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

func (g *LeakGuard) SupportedExtensions() []string     { return []string{".go"} }
func (g *LeakGuard) UnobservedFiles() []string         { return nil }
func (g *LeakGuard) MainInputFileRelativePath() string { return "" }

func (g *LeakGuard) NewFileEvent(fileName, extension, filePath, event string) error {
	if fileName != watchedFileName {
		return nil
	}

	info, err := os.Stat(filePath)
	if err != nil {
		return nil
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

func checkFile(filePath string) error {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filePath, nil, parser.ImportsOnly|parser.ParseComments)
	if err != nil {
		return nil
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

	return fmt.Err(
		filePath + " imports \"" + spriteImportPath + "\" without a leading \"//go:build !wasm\" constraint — " +
			"this ships sprite geometry and encoding/json into the WASM binary. " +
			"Add \"//go:build !wasm\" as the file's first line.",
	)
}

func hasWasmExclusionTag(f *ast.File) bool {
	for _, group := range f.Comments {
		if group.Pos() < f.Package {
			for _, comment := range group.List {
				text := comment.Text
				if (strings.Contains(text, "//go:build") || strings.Contains(text, "// +build")) &&
					strings.Contains(text, "!wasm") {
					return true
				}
			}
		}
	}
	return false
}
