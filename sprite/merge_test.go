package sprite

import "testing"

func spriteWith(ids ...string) *Sprite {
	s := NewSprite()
	for _, id := range ids {
		s.AddRaw(id, "<path/>", "0 0 24 24")
	}
	return s
}

// The old Merge concatenated without deduplicating, so two modules declaring the
// same icon emitted two <symbol> elements with the same id — invalid HTML.
// assetmin compensated downstream with its own first-wins policy, which this
// package knew nothing about.
func TestMergeDeduplicatesKeepingFirst(t *testing.T) {
	a := spriteWith("home", "user")
	b := spriteWith("user", "gear")

	got := MergeAll(a, b).IDs()
	want := []string{"home", "user", "gear"}

	if len(got) != len(want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected %v, got %v", want, got)
		}
	}
}

// The old Merge appended in place, permanently mutating the receiver — which in
// the daemon was a cached sprite shared across every extraction.
func TestMergeDoesNotMutateOperands(t *testing.T) {
	a := spriteWith("home", "user")
	b := spriteWith("gear")

	_ = MergeAll(a, b)

	if a.Len() != 2 {
		t.Errorf("Merge mutated the receiver: expected 2 icons, got %d", a.Len())
	}
	if b.Len() != 1 {
		t.Errorf("Merge mutated the argument: expected 1 icon, got %d", b.Len())
	}
}

// The old Merge returned `other` unchanged when the receiver was nil, so the
// caller believed it held a fresh sprite while actually holding the first
// package's cached one. The next merge then corrupted that cache.
func TestMergeOnNilReceiverDoesNotAlias(t *testing.T) {
	b := spriteWith("gear")

	got := MergeAll(nil, b)

	if got == b {
		t.Fatal("Merge returned the argument's pointer; the caller can now corrupt a cached sprite")
	}
	got.AddRaw("extra", "<path/>", "0 0 24 24")
	if b.Len() != 1 {
		t.Errorf("mutating the result changed the argument: expected 1 icon, got %d", b.Len())
	}
}

// TestMergeAllIsTheOnlyCombiner fija la razón de que Sprite.Merge no exista.
// Era una función pura con forma de método: x.Merge(y) compila como sentencia
// y tira el resultado sin aviso. Esa exacta llamada, en tinywasm/sitec, dejó
// el sprite vacío y con él todos los iconos del ecosistema.
func TestMergeAllIsTheOnlyCombiner(t *testing.T) {
	a := NewSprite(Define("a", "0 0 1 1", Path("M0 0")))
	b := NewSprite(Define("b", "0 0 1 1", Path("M1 1")))

	got := MergeAll(a, b)

	if got.Len() != 2 {
		t.Fatalf("MergeAll debe combinar ambos, obtuve %d", got.Len())
	}
	if a.Len() != 1 || b.Len() != 1 {
		t.Errorf("MergeAll no debe mutar sus argumentos: a=%d b=%d", a.Len(), b.Len())
	}
}

func TestMergeAllIgnoresNilAndKeepsOrder(t *testing.T) {
	got := MergeAll(nil, spriteWith("a", "b"), nil, spriteWith("b", "c")).IDs()
	want := []string{"a", "b", "c"}

	if len(got) != len(want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected %v, got %v", want, got)
		}
	}
}

func TestMergeAllWithNothingReturnsEmptyNonNil(t *testing.T) {
	got := MergeAll()
	if got == nil {
		t.Fatal("MergeAll must always return a usable sprite")
	}
	if got.Len() != 0 {
		t.Errorf("expected no icons, got %d", got.Len())
	}
}

// assetmin patched the empty case after the fact, reinjecting the wrapper by
// hand so the document kept a stable injection point. That markup is this
// package's to emit.
func TestEmptySpriteRendersWrapper(t *testing.T) {
	if got := NewSprite().String(); got != EmptyWrapper {
		t.Errorf("expected %q, got %q", EmptyWrapper, got)
	}
}

func TestNilSpriteRendersNothing(t *testing.T) {
	if got := (*Sprite)(nil).String(); got != "" {
		t.Errorf("a nil sprite must render nothing, got %q", got)
	}
}

// assetmin.checkIconID rendered the whole sprite to a string and looked for
// `id="..."` as a substring, so an icon whose BODY mentioned another id gave a
// false positive.
func TestHasDoesNotMatchRenderedMarkup(t *testing.T) {
	s := NewSprite()
	s.AddRaw("real", `<path id="decoy"/>`, "0 0 24 24")

	if !s.Has("real") {
		t.Error("Has must find an icon by its ID")
	}
	if s.Has("decoy") {
		t.Error("Has matched an ID appearing inside a body; it must not resolve by rendered text")
	}
	if (*Sprite)(nil).Has("real") {
		t.Error("a nil sprite contains nothing")
	}
}
