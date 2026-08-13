package sprite

// EmptyWrapper is the markup a sprite with zero icons renders as. The document
// that injects a sprite needs a stable insertion point, so an empty sprite must
// still emit its wrapper. Composing this outside forces every consumer to
// duplicate the markup — which is exactly what assetmin used to do.
const EmptyWrapper = `<svg aria-hidden="true" style="display:none"></svg>`

// MergeAll merges sprites in order, keeping the first occurrence of each icon
// ID. This is the single home of the ecosystem's deduplication policy:
// consumers merging per package or per module call here instead of
// reimplementing it.
//
// The caller decides the ORDER (sitec sorts by module name so the result is
// stable across scans); this function owns the POLICY.
//
// nil entries are ignored. The result is always a fresh, non-nil *Sprite: no
// argument is mutated and no argument's pointer is returned.
func MergeAll(sprites ...*Sprite) *Sprite {
	seen := make(map[string]bool)
	var deduped []Definition

	for _, s := range sprites {
		if s == nil {
			continue
		}
		for _, def := range s.icons {
			id := def.Icon.ID()
			if seen[id] {
				continue
			}
			seen[id] = true
			deduped = append(deduped, def)
		}
	}

	return &Sprite{icons: deduped}
}

// Has reports whether the sprite already contains an icon with that ID.
// A nil sprite contains nothing.
func (s *Sprite) Has(id string) bool {
	if s == nil {
		return false
	}
	for _, def := range s.icons {
		if def.Icon.ID() == id {
			return true
		}
	}
	return false
}

// IDs returns the icon IDs held, in insertion order. A nil sprite returns nil.
func (s *Sprite) IDs() []string {
	if s == nil {
		return nil
	}
	out := make([]string, 0, len(s.icons))
	for _, def := range s.icons {
		out = append(out, def.Icon.ID())
	}
	return out
}
