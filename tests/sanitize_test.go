//go:build !wasm

package svg_test

import (
	"bytes"
	"encoding/xml"
	"strings"
	"testing"

	"webtyp.com/svg/sanitize"
)

func TestCleanRejectsScript(t *testing.T) {
	src := []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`)
	_, err := sanitize.Clean(src)
	if err == nil {
		t.Fatalf("expected error for <script>")
	}
	if !strings.Contains(err.Error(), "<script>") {
		t.Fatalf("error should name <script>, got %q", err.Error())
	}
	want := "svg: el archivo contiene <script>: un logo no ejecuta codigo"
	if err.Error() != want {
		t.Fatalf("unexpected error %q want %q", err.Error(), want)
	}
}

func TestCleanRejectsEventHandler(t *testing.T) {
	src := []byte(`<svg onload="alert(1)"><path d="M0 0"/></svg>`)
	_, err := sanitize.Clean(src)
	if err == nil {
		t.Fatalf("expected error for onload")
	}
	if !strings.Contains(err.Error(), "onload") {
		t.Fatalf("error should name onload, got %q", err.Error())
	}
	want := "svg: el archivo contiene el manejador onload: un logo no ejecuta codigo"
	if err.Error() != want {
		t.Fatalf("unexpected error %q want %q", err.Error(), want)
	}
}

func TestCleanRejectsForeignObject(t *testing.T) {
	src := []byte(`<svg><foreignObject><div>hi</div></foreignObject></svg>`)
	_, err := sanitize.Clean(src)
	if err == nil {
		t.Fatalf("expected error for <foreignObject>")
	}
	if !strings.Contains(err.Error(), "foreignObject") {
		t.Fatalf("error should name foreignObject, got %q", err.Error())
	}
	want := "svg: el archivo contiene <foreignObject>: un logo no incrusta HTML"
	if err.Error() != want {
		t.Fatalf("unexpected error %q want %q", err.Error(), want)
	}
}

func TestCleanStripsStyle(t *testing.T) {
	src := []byte(`<svg><style>@import url(https://evil)</style><path d="M0 0" fill="red"/></svg>`)
	out, err := sanitize.Clean(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(string(out), "<style") {
		t.Fatalf("style should be stripped, got %s", string(out))
	}
	if strings.Contains(string(out), "@import") {
		t.Fatalf("style content should be stripped, got %s", string(out))
	}
	if !strings.Contains(string(out), "<path") {
		t.Fatalf("path should survive, got %s", string(out))
	}
}

func TestCleanStripsExternalHref(t *testing.T) {
	src := []byte(`<svg><use href="https://evil/x"></use><use href="#logo"></use></svg>`)
	out, err := sanitize.Clean(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(string(out), "https://evil") {
		t.Fatalf("external href should be stripped, got %s", string(out))
	}
	if !strings.Contains(string(out), `href="#logo"`) {
		t.Fatalf("local href should be kept, got %s", string(out))
	}
	// also xlink variant external should be stripped, local kept
	src2 := []byte(`<svg xmlns:xlink="http://www.w3.org/1999/xlink"><use xlink:href="https://evil"></use><use xlink:href="#a"></use></svg>`)
	out2, err := sanitize.Clean(src2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(string(out2), "https://evil") {
		t.Fatalf("external xlink href should be stripped, got %s", string(out2))
	}
	if !strings.Contains(string(out2), "#a") {
		t.Fatalf("local xlink href should be kept, got %s", string(out2))
	}
}

func TestCleanStripsUnknownAttributes(t *testing.T) {
	src := []byte(`<svg><path d="M0 0" inkscape:label="foo" fill="red"/></svg>`)
	out, err := sanitize.Clean(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(string(out), "inkscape") {
		t.Fatalf("unknown attr should be stripped, got %s", string(out))
	}
	if !strings.Contains(string(out), `d="M0 0"`) {
		t.Fatalf("d should survive, got %s", string(out))
	}
	if !strings.Contains(string(out), `fill="red"`) {
		t.Fatalf("fill should survive, got %s", string(out))
	}
}

func TestCleanKeepsRealLogo(t *testing.T) {
	src := []byte(`<svg viewBox="0 0 10 10"><path d="M0 0"/><path d="M1 1"/><linearGradient id="g"><stop offset="0" stop-color="red"/></linearGradient></svg>`)
	out, err := sanitize.Clean(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(string(out), `viewBox="0 0 10 10"`) {
		t.Fatalf("viewBox should be kept, got %s", string(out))
	}
	if count(string(out), "<path") != 2 {
		t.Fatalf("expected 2 paths, got %s", string(out))
	}
	if !strings.Contains(string(out), "linearGradient") {
		t.Fatalf("linearGradient should survive, got %s", string(out))
	}
	if !strings.Contains(string(out), "<stop") {
		t.Fatalf("stop should survive, got %s", string(out))
	}
	// still valid XML with svg root
	var v struct {
		XMLName xml.Name `xml:"svg"`
	}
	if err := xml.NewDecoder(bytes.NewReader(out)).Decode(&v); err != nil {
		t.Fatalf("output should be valid XML: %v out=%s", err, string(out))
	}
	if v.XMLName.Local != "svg" {
		t.Fatalf("root should be svg, got %q", v.XMLName.Local)
	}
}

func count(s, sub string) int {
	return strings.Count(s, sub)
}

func TestCleanIsIdempotent(t *testing.T) {
	src := []byte(`<svg viewBox="0 0 10 10"><path d="M0 0"/><path d="M1 1"/><linearGradient id="g"><stop offset="0" stop-color="red"/></linearGradient></svg>`)
	first, err := sanitize.Clean(src)
	if err != nil {
		t.Fatalf("first clean failed: %v", err)
	}
	second, err := sanitize.Clean(first)
	if err != nil {
		t.Fatalf("second clean failed: %v", err)
	}
	if !bytes.Equal(first, second) {
		t.Fatalf("not idempotent:\nfirst=%s\nsecond=%s", string(first), string(second))
	}
}

func TestCleanRejectsGarbage(t *testing.T) {
	_, err := sanitize.Clean([]byte(`not xml`))
	if err == nil {
		t.Fatalf("expected error for not xml")
	}
	if err.Error() != "svg: el archivo no es un SVG valido" {
		t.Fatalf("unexpected error %q", err.Error())
	}
	_, err = sanitize.Clean([]byte(`<html><body>hi</body></html>`))
	if err == nil {
		t.Fatalf("expected error for non-svg xml")
	}
	if err.Error() != "svg: el archivo no es un SVG valido" {
		t.Fatalf("unexpected error %q", err.Error())
	}
	_, err = sanitize.Clean([]byte(``))
	if err == nil {
		t.Fatalf("expected error for empty")
	}
}
