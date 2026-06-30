package svg

import "github.com/tinywasm/dom"

// node es markup interno del cuerpo de un símbolo.
type node string

// Path renderiza <path fill="currentColor" d="<d>"/>.
func Path(d string) node {
	return node(`<path fill="currentColor" d="` + d + `"/>`)
}

// Raw es el escape hatch para markup que Path no expresa.
func Raw(s string) node {
	return node(s)
}

// Icon es una definición tipada de icono: id + viewBox + cuerpo.
type Icon struct {
	id      string
	viewBox string
	body    string
}

// Define crea un icono reutilizable.
func Define(id, viewBox string, body ...node) Icon {
	var content string
	for _, n := range body {
		content += string(n)
	}
	return Icon{id: id, viewBox: viewBox, body: content}
}

// ID devuelve el id del símbolo.
func (i Icon) ID() string {
	return i.id
}

// Render emite <svg aria-hidden focusable=false class=...><use href="#id"/></svg>.
func (i Icon) Render(classes ...string) *dom.Element {
	el := dom.NewElement("svg").
		Attr("aria-hidden", "true").
		Attr("focusable", "false").
		Child(Use().Attr("href", "#"+i.id))

	// Agregar clases si existen
	for _, className := range classes {
		el.Class(className)
	}

	return el
}

// NewSprite construye un *Sprite a partir de iconos tipados.
func NewSprite(icons ...Icon) *Sprite {
	s := &Sprite{}
	for _, icon := range icons {
		s.icons = append(s.icons, iconEntry{
			id:      icon.id,
			content: icon.body,
			viewBox: icon.viewBox,
		})
	}
	return s
}
