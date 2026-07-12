package svg_test

import (
	"strings"
	"testing"

	"github.com/tinywasm/svg"
)

func TestIcon_Render(t *testing.T) {
	icon := svg.Icon("home")
	if icon.ID() != "home" {
		t.Errorf("expected ID 'home', got %q", icon.ID())
	}

	el := icon.Render("nav-icon")
	html := el.String()

	if !strings.Contains(html, "aria-hidden='true'") {
		t.Error("expected aria-hidden")
	}
	if !strings.Contains(html, "focusable='false'") {
		t.Error("expected focusable")
	}
	if !strings.Contains(html, "class='nav-icon'") {
		t.Error("expected class")
	}
	if !strings.Contains(html, "href='#home'") {
		t.Error("expected href")
	}
}
