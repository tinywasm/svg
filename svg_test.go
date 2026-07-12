package svg_test

import (
	"strings"
	"testing"

	"github.com/tinywasm/svg"
	"github.com/tinywasm/svg/sprite"
)

func TestIconLegacy_Structure(t *testing.T) {
	got := svg.IconLegacy("home", "nav-icon").String()
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
	s := sprite.New().
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

func TestPath(t *testing.T) {
	got := string(sprite.Path("M1 1"))
	want := `<path fill="currentColor" d="M1 1"/>`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestIcon_Render(t *testing.T) {
	iconID := svg.Icon("home")
	_ = sprite.Define(iconID, "0 0 20 20", sprite.Path("M1 1"))
	if iconID.ID() != "home" {
		t.Errorf("expected ID home, got %s", iconID.ID())
	}

	got := iconID.Render("custom-class").String()
	if !strings.Contains(got, `href='#home'`) {
		t.Error("expected href")
	}
	if !strings.Contains(got, `class='custom-class'`) {
		t.Error("expected class")
	}
}

func TestNewSprite(t *testing.T) {
	iconHome := sprite.Define(svg.Icon("home"), "0 0 576 512", sprite.Path("M1 1"))
	iconInfo := sprite.Define(svg.Icon("info"), "0 0 16 16", sprite.Path("m7 11h2v2h-2z"))

	s := sprite.NewSprite(iconHome, iconInfo)
	out := s.String()

	if !strings.Contains(out, `id="home"`) {
		t.Error("expected home symbol")
	}
	if !strings.Contains(out, `viewBox="0 0 576 512"`) {
		t.Error("expected custom viewBox")
	}
	if !strings.Contains(out, `id="info"`) {
		t.Error("expected info symbol")
	}
	if !strings.Contains(out, `viewBox="0 0 16 16"`) {
		t.Error("expected info viewBox")
	}
	if !strings.Contains(out, `fill="currentColor"`) {
		t.Error("expected currentColor in body")
	}
}

func TestSprite_Merge(t *testing.T) {
	a := sprite.New().Add("icon-a", "<path/>")
	b := sprite.New().Add("icon-b", "<path/>")
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
	var s *sprite.Sprite
	if s.String() != "" {
		t.Fatal("nil Sprite.String() should be empty")
	}
	if s.Merge(sprite.New()) == nil {
		t.Fatal("nil Merge should not panic")
	}
}
