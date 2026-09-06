package sprite_test

import (
	"strings"
	"testing"

	"webtyp.com/svg"
	"webtyp.com/svg/sprite"
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

// TestSprite_Icons guards the typed accessor consumers (e.g. assetmin, merging
// several modules' sprites) use instead of round-tripping through String() and
// re-parsing markup.
func TestSprite_Icons_NilReceiver(t *testing.T) {
	var nilSprite *sprite.Sprite
	if defs := nilSprite.Icons(); defs != nil {
		t.Fatalf("nil-receiver Icons() = %v, want nil", defs)
	}
}

func TestSprite_IconsReturnsTypedFields(t *testing.T) {
	sp := sprite.NewSprite(
		sprite.Define(svg.Icon("home"), "0 0 576 512", sprite.Path("M1 1")),
		sprite.Define(svg.Icon("info"), "0 0 16 16", sprite.Path("M2 2")),
	)

	defs := sp.Icons()
	if len(defs) != 2 {
		t.Fatalf("Icons() len = %d, want 2", len(defs))
	}
	if defs[0].Icon.ID() != "home" || defs[0].ViewBox != "0 0 576 512" {
		t.Errorf("Icons()[0] = %+v, want id=home viewBox=\"0 0 576 512\"", defs[0])
	}
	if defs[1].Icon.ID() != "info" || defs[1].ViewBox != "0 0 16 16" {
		t.Errorf("Icons()[1] = %+v, want id=info viewBox=\"0 0 16 16\"", defs[1])
	}

	// Mutating the returned slice must not affect the sprite's own state (Icons
	// returns a copy).
	defs[0] = sprite.Definition{}
	if sp.Icons()[0].Icon.ID() != "home" {
		t.Error("Icons() must return a defensive copy, not the internal slice")
	}
}
