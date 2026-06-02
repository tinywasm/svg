package svg_test

import (
	"encoding/json"
	"testing"

	"github.com/tinywasm/svg"
)

func TestSprite_JSONRoundTrip(t *testing.T) {
	original := svg.NewSprite(
		svg.Define("home", "0 0 24 24", svg.Path("M1 1")),
		svg.Define("rect", "0 0 16 16", svg.Raw(`<rect x="0" y="0" width="10" height="10"/>`)),
	)

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	recovered := &svg.Sprite{}
	if err := json.Unmarshal(data, recovered); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if original.String() != recovered.String() {
		t.Errorf("Round-trip failed.\nOriginal:  %s\nRecovered: %s", original.String(), recovered.String())
	}
}

func TestSprite_JSONEmpty(t *testing.T) {
	s := &svg.Sprite{}
	data, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	if string(data) != "[]" {
		t.Errorf("Expected empty array [], got %s", string(data))
	}

	recovered := &svg.Sprite{}
	if err := json.Unmarshal(data, recovered); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if recovered.String() != "" {
		t.Errorf("Expected empty string, got %s", recovered.String())
	}
}

func TestSprite_JSONNil(t *testing.T) {
	var s *svg.Sprite
	data, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	if string(data) != "null" {
		t.Errorf("Expected null, got %s", string(data))
	}

	// Unmarshaling into nil *Sprite should return an error as per implementation
	err = json.Unmarshal([]byte("[]"), s)
	if err == nil {
		t.Error("Expected error when unmarshaling into nil *Sprite, got nil")
	}
}
