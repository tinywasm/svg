package sprite_test

import (
	"strings"
	"testing"

	"github.com/tinywasm/svg"
	"github.com/tinywasm/svg/sprite"
)

func TestSprite_String(t *testing.T) {
	iconHome := svg.Icon("home")
	iconInfo := svg.Icon("info")

	s := sprite.NewSprite(
		sprite.Define(iconHome, "0 0 576 512", sprite.Path("M1 1")),
		sprite.Define(iconInfo, "0 0 16 16", sprite.Path("M2 2")),
	)

	out := s.String()
	if !strings.Contains(out, `id="home"`) {
		t.Error("expected home symbol")
	}
	if !strings.Contains(out, `viewBox="0 0 576 512"`) {
		t.Error("expected custom viewBox")
	}
	if !strings.Contains(out, `fill="currentColor"`) {
		t.Error("expected fill=\"currentColor\"")
	}
}

func TestSprite_JSON(t *testing.T) {
	s := sprite.NewSprite(
		sprite.Define(svg.Icon("home"), "0 0 20 20", sprite.Path("M1 1")),
	)

	encoded, err := s.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON failed: %v", err)
	}

	want := `[{"id":"home","content":"<path fill=\"currentColor\" d=\"M1 1\"/>","viewBox":"0 0 20 20"}]`
	if string(encoded) != want {
		t.Errorf("got %s, want %s", string(encoded), want)
	}

	var s2 sprite.Sprite
	if err := s2.UnmarshalJSON(encoded); err != nil {
		t.Fatalf("UnmarshalJSON failed: %v", err)
	}

	if s2.String() != s.String() {
		t.Errorf("roundtrip failed: got %s, want %s", s2.String(), s.String())
	}
}
