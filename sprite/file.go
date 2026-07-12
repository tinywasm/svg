package sprite

import "github.com/tinywasm/fmt"

// AddFile adds a whole .svg file (as read from disk) to the sprite as a <symbol>.
// The viewBox is taken from the file's root <svg> element, and only its inner
// markup becomes the symbol body — keeping the root tag would nest an <svg>
// inside the <symbol>, which is not valid sprite markup.
//
// A file with no viewBox is rejected: nothing can recover the coordinate system
// the drawing was authored in, and defaulting to one clips or misaligns it.
func (s *Sprite) AddFile(id, svgFile string) error {
	viewBox, ok := viewBoxOf(svgFile)
	if !ok {
		return fmt.Err("svg file has no viewBox attribute:", id)
	}
	body, ok := innerOf(svgFile)
	if !ok {
		return fmt.Err("svg file has no root <svg> element:", id)
	}
	s.AddRaw(id, body, viewBox)
	return nil
}

// viewBoxOf reads the value of the viewBox attribute.
func viewBoxOf(s string) (string, bool) {
	i := index(s, 0, "viewBox")
	if i < 0 {
		return "", false
	}
	i = skipSpace(s, i+len("viewBox"))
	if i >= len(s) || s[i] != '=' {
		return "", false
	}
	i = skipSpace(s, i+1)
	if i >= len(s) || (s[i] != '"' && s[i] != '\'') {
		return "", false
	}
	quote := s[i]
	i++
	j := i
	for j < len(s) && s[j] != quote {
		j++
	}
	if j >= len(s) || j == i {
		return "", false
	}
	return s[i:j], true
}

// innerOf returns the markup between the root <svg ...> and its closing </svg>.
func innerOf(s string) (string, bool) {
	open := index(s, 0, "<svg")
	if open < 0 {
		return "", false
	}
	gt := index(s, open, ">")
	if gt < 0 {
		return "", false
	}
	end := lastIndex(s, "</svg>")
	if end < gt {
		return "", false
	}
	return s[gt+1 : end], true
}

func skipSpace(s string, i int) int {
	for i < len(s) && (s[i] == ' ' || s[i] == '\t' || s[i] == '\n' || s[i] == '\r') {
		i++
	}
	return i
}

func index(s string, from int, sub string) int {
	for i := from; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func lastIndex(s, sub string) int {
	for i := len(s) - len(sub); i >= 0; i-- {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
