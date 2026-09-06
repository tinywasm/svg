package watch_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"webtyp.com/svg/watch"
)

func TestLeakGuard(t *testing.T) {
	tmp := t.TempDir()

	t.Run("MissingTag", func(t *testing.T) {
		g := watch.New()
		path := filepath.Join(tmp, "svg.go")
		content := `package main
import "webtyp.com/svg/sprite"
`
		os.WriteFile(path, []byte(content), 0644)
		err := g.NewFileEvent("svg.go", ".go", path, "save")
		if err == nil {
			t.Error("expected error for missing tag")
		}
	})

	t.Run("CorrectTag", func(t *testing.T) {
		g := watch.New()
		path := filepath.Join(tmp, "svg.go")
		content := `//go:build !wasm
package main
import "webtyp.com/svg/sprite"
`
		os.WriteFile(path, []byte(content), 0644)
		err := g.NewFileEvent("svg.go", ".go", path, "save")
		if err != nil {
			t.Errorf("expected no error for correct tag, got %v", err)
		}
	})

	t.Run("NoSpriteImport", func(t *testing.T) {
		g := watch.New()
		path := filepath.Join(tmp, "svg.go")
		content := `package main
import "webtyp.com/svg"
`
		os.WriteFile(path, []byte(content), 0644)
		err := g.NewFileEvent("svg.go", ".go", path, "save")
		if err != nil {
			t.Errorf("expected no error when sprite is not imported, got %v", err)
		}
	})

	t.Run("IgnoresOtherFiles", func(t *testing.T) {
		g := watch.New()
		path := filepath.Join(tmp, "main.go")
		content := `package main
import "webtyp.com/svg/sprite"
`
		os.WriteFile(path, []byte(content), 0644)
		err := g.NewFileEvent("main.go", ".go", path, "save")
		if err != nil {
			t.Errorf("expected no error for non-svg.go file, got %v", err)
		}
	})

	t.Run("MtimeCache", func(t *testing.T) {
		g := watch.New()
		path := filepath.Join(tmp, "svg.go")
		content := `package main
import "webtyp.com/svg/sprite"
`
		os.WriteFile(path, []byte(content), 0644)
		// Set a specific mtime
		mtime := time.Now().Truncate(time.Second)
		os.Chtimes(path, mtime, mtime)

		err1 := g.NewFileEvent("svg.go", ".go", path, "save")
		if err1 == nil {
			t.Fatal("expected error")
		}

		// Change file content but keep mtime same (force cache hit)
		os.WriteFile(path, []byte("//go:build !wasm\npackage main\nimport \"webtyp.com/svg/sprite\""), 0644)
		os.Chtimes(path, mtime, mtime)

		err2 := g.NewFileEvent("svg.go", ".go", path, "save")
		if err2 != err1 {
			t.Errorf("expected cached error, got %v", err2)
		}
	})
}
