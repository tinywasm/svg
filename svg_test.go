package svg_test

import (
	"strings"
	"testing"

	. "github.com/tinywasm/svg"
)

func TestIcon_Structure(t *testing.T) {
	got := Icon("home", "nav-icon").String()
	if !strings.Contains(got, `href='#home'`) {
		t.Error("expected href")
	}
	if !strings.Contains(got, `aria-hidden='true'`) {
		t.Error("expected aria-hidden")
	}
	if !strings.Contains(got, `class='nav-icon'`) {
		t.Error("expected class")
	}
}

func TestSprite_String(t *testing.T) {
	s := New().
		Add("home", `<path fill="currentColor" d="M1 1"/>`, "0 0 576 512").
		Add("info", `<path fill="currentColor" d="m7 11h2v2h-2z"/>`)

	out := s.String()
	if !strings.Contains(out, `id="home"`) {
		t.Error("expected home symbol")
	}
	if !strings.Contains(out, `viewBox="0 0 576 512"`) {
		t.Error("expected custom viewBox")
	}
	if !strings.Contains(out, `viewBox="0 0 16 16"`) {
		t.Error("expected default viewBox")
	}
}

func TestSprite_Merge(t *testing.T) {
	a := New().Add("icon-a", "<path/>")
	b := New().Add("icon-b", "<path/>")
	a.Merge(b)
	out := a.String()
	if !strings.Contains(out, `id="icon-a"`) {
		t.Error("expected icon-a")
	}
	if !strings.Contains(out, `id="icon-b"`) {
		t.Error("expected icon-b")
	}
}

func TestSprite_Nil_Safe(t *testing.T) {
	var s *Sprite
	if s.String() != "" {
		t.Fatal("nil Sprite.String() should be empty")
	}
	if s.Merge(New()) == nil {
		t.Fatal("nil Merge should not panic")
	}
}
