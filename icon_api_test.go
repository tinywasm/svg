package svg_test

import (
	"strings"
	"testing"

	"github.com/tinywasm/svg"
)

func TestPath_Output(t *testing.T) {
	n := svg.Path("M280 148")
	expected := `<path fill="currentColor" d="M280 148"/>`
	if string(n) != expected {
		t.Errorf("expected %q, got %q", expected, string(n))
	}
}

func TestRaw_Output(t *testing.T) {
	n := svg.Raw(`<circle cx="16" cy="16" r="16"/>`)
	expected := `<circle cx="16" cy="16" r="16"/>`
	if string(n) != expected {
		t.Errorf("expected %q, got %q", expected, string(n))
	}
}

func TestDefine_ID(t *testing.T) {
	icon := svg.Define("home", "0 0 576 512", svg.Path("M280 148"))
	if icon.ID() != "home" {
		t.Errorf("expected ID 'home', got %q", icon.ID())
	}
}

func TestIcon_Render(t *testing.T) {
	icon := svg.Define("home", "0 0 576 512", svg.Path("M280 148"))
	el := icon.Render("nav-icon")

	html := el.String()
	if !strings.Contains(html, `aria-hidden='true'`) {
		t.Error("expected aria-hidden attribute")
	}
	if !strings.Contains(html, `focusable='false'`) {
		t.Error("expected focusable attribute")
	}
	if !strings.Contains(html, `class='nav-icon'`) {
		t.Error("expected class attribute")
	}
	if !strings.Contains(html, `href='#home'`) {
		t.Error("expected href to home")
	}
}

func TestIcon_RenderMultipleClasses(t *testing.T) {
	icon := svg.Define("home", "0 0 576 512", svg.Path("M280 148"))
	el := icon.Render("nav-icon", "large")

	html := el.String()
	if !strings.Contains(html, `nav-icon`) {
		t.Error("expected first class")
	}
	if !strings.Contains(html, `large`) {
		t.Error("expected second class")
	}
}

func TestNewSprite_SingleIcon(t *testing.T) {
	icon := svg.Define("home", "0 0 576 512", svg.Path("M280 148"))
	sprite := svg.NewSprite(icon)

	html := sprite.String()
	if !strings.Contains(html, `<symbol id="home"`) {
		t.Error("expected home symbol")
	}
	if !strings.Contains(html, `viewBox="0 0 576 512"`) {
		t.Error("expected custom viewBox")
	}
	if !strings.Contains(html, `<path fill="currentColor" d="M280 148"/>`) {
		t.Error("expected path content")
	}
}

func TestNewSprite_MultipleIcons(t *testing.T) {
	home := svg.Define("home", "0 0 576 512", svg.Path("M280 148"))
	info := svg.Define("info", "0 0 16 16", svg.Path("m7 11h2v2h-2z"))
	sprite := svg.NewSprite(home, info)

	html := sprite.String()
	if !strings.Contains(html, `id="home"`) {
		t.Error("expected home symbol")
	}
	if !strings.Contains(html, `id="info"`) {
		t.Error("expected info symbol")
	}
	if !strings.Contains(html, `viewBox="0 0 576 512"`) {
		t.Error("expected home viewBox")
	}
	if !strings.Contains(html, `viewBox="0 0 16 16"`) {
		t.Error("expected info viewBox")
	}
}

func TestNewSprite_WithRaw(t *testing.T) {
	icon := svg.Define("logo", "0 0 32 32",
		svg.Raw(`<circle cx="16" cy="16" r="16"/>`),
	)
	sprite := svg.NewSprite(icon)

	html := sprite.String()
	if !strings.Contains(html, `<circle cx="16" cy="16" r="16"/>`) {
		t.Error("expected circle content from Raw")
	}
}

func TestDefine_MultipleNodes(t *testing.T) {
	icon := svg.Define("multi", "0 0 100 100",
		svg.Path("M10 10"),
		svg.Path("M20 20"),
	)
	sprite := svg.NewSprite(icon)

	html := sprite.String()
	if !strings.Contains(html, `<path fill="currentColor" d="M10 10"/>`) {
		t.Error("expected first path")
	}
	if !strings.Contains(html, `<path fill="currentColor" d="M20 20"/>`) {
		t.Error("expected second path")
	}
}

func TestSprite_Merge(t *testing.T) {
	a := svg.NewSprite(
		svg.Define("icon-a", "0 0 16 16", svg.Path("M1 1")),
	)
	b := svg.NewSprite(
		svg.Define("icon-b", "0 0 16 16", svg.Path("M2 2")),
	)

	a.Merge(b)
	html := a.String()
	if !strings.Contains(html, `id="icon-a"`) {
		t.Error("expected icon-a")
	}
	if !strings.Contains(html, `id="icon-b"`) {
		t.Error("expected icon-b")
	}
}

func TestSprite_Nil_Safe(t *testing.T) {
	var s *svg.Sprite
	if s.String() != "" {
		t.Fatal("nil Sprite.String() should be empty")
	}
	if s.Merge(svg.NewSprite()) == nil {
		t.Fatal("nil Merge should not panic")
	}
}
