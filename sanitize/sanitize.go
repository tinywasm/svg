//go:build !wasm

package sanitize

import (
	"bytes"
	"encoding/xml"
	"io"
	"strings"

	"github.com/tinywasm/fmt"
)

var allowedElements = []string{
	"svg",
	"g",
	"defs",
	"symbol",
	"use",
	"title",
	"desc",
	"path",
	"rect",
	"circle",
	"ellipse",
	"line",
	"polyline",
	"polygon",
	"lineargradient",
	"radialgradient",
	"stop",
	"clippath",
	"mask",
	"text",
	"tspan",
}

var allowedAttrs = []string{
	"xmlns",
	"xmlns:xlink",
	"viewbox",
	"width",
	"height",
	"preserveaspectratio",
	"d",
	"points",
	"x",
	"y",
	"x1",
	"y1",
	"x2",
	"y2",
	"cx",
	"cy",
	"r",
	"rx",
	"ry",
	"fill",
	"fill-opacity",
	"fill-rule",
	"stroke",
	"stroke-width",
	"stroke-linecap",
	"stroke-linejoin",
	"stroke-dasharray",
	"stroke-opacity",
	"opacity",
	"transform",
	"gradientunits",
	"gradienttransform",
	"offset",
	"stop-color",
	"stop-opacity",
	"clip-path",
	"mask",
	"class",
	"id",
	"role",
	"aria-label",
}

func isAllowedElement(lower string) bool {
	for _, e := range allowedElements {
		if e == lower {
			return true
		}
	}
	return false
}

func isAllowedAttr(lower string) bool {
	for _, a := range allowedAttrs {
		if a == lower {
			return true
		}
	}
	return false
}

type node struct {
	name    string
	attrs   []xml.Attr
	content []content
}

type content struct {
	isText bool
	text   string
	elem   *node
}

// Clean devuelve el SVG sin nada ejecutable ni nada que salga a la red.
// Elimina lo que no entiende y falla ante lo que sólo puede ser un ataque.
func Clean(src []byte) ([]byte, error) {
	if len(src) == 0 {
		return nil, fmt.Err("svg: el archivo no es un SVG valido")
	}
	dec := xml.NewDecoder(bytes.NewReader(src))
	dec.Strict = true

	for {
		tok, err := dec.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Err("svg: el archivo no es un SVG valido")
		}
		switch t := tok.(type) {
		case xml.StartElement:
			low := strings.ToLower(t.Name.Local)
			if low != "svg" {
				return nil, fmt.Err("svg: el archivo no es un SVG valido")
			}
			contents, err := sanitizeElement(dec, t)
			if err != nil {
				return nil, err
			}
			if len(contents) == 0 {
				return nil, fmt.Err("svg: el archivo no es un SVG valido")
			}
			// contents should contain single svg node
			var root *node
			for _, c := range contents {
				if !c.isText && c.elem != nil && strings.ToLower(c.elem.name) == "svg" {
					root = c.elem
					break
				}
			}
			if root == nil {
				// if sanitize filtered root but kept children, still need svg
				return nil, fmt.Err("svg: el archivo no es un SVG valido")
			}
			var buf bytes.Buffer
			writeNode(&buf, root)
			return buf.Bytes(), nil
		case xml.CharData:
			if strings.TrimSpace(string(t)) != "" {
				return nil, fmt.Err("svg: el archivo no es un SVG valido")
			}
		case xml.Comment, xml.Directive, xml.ProcInst:
			continue
		case xml.EndElement:
			continue
		}
	}
	return nil, fmt.Err("svg: el archivo no es un SVG valido")
}

func sanitizeElement(dec *xml.Decoder, start xml.StartElement) ([]content, error) {
	low := strings.ToLower(start.Name.Local)

	if low == "script" {
		return nil, fmt.Err("svg: el archivo contiene <script>: un logo no ejecuta codigo")
	}
	if low == "foreignobject" {
		return nil, fmt.Err("svg: el archivo contiene <foreignObject>: un logo no incrusta HTML")
	}
	for _, a := range start.Attr {
		alow := strings.ToLower(a.Name.Local)
		if strings.HasPrefix(alow, "on") {
			return nil, fmt.Err("svg: el archivo contiene el manejador " + alow + ": un logo no ejecuta codigo")
		}
	}

	if low == "style" {
		depth := 1
		for depth > 0 {
			tok, err := dec.Token()
			if err != nil {
				return nil, fmt.Err("svg: el archivo no es un SVG valido")
			}
			switch tt := tok.(type) {
			case xml.StartElement:
				if strings.EqualFold(tt.Name.Local, "style") {
					depth++
				}
			case xml.EndElement:
				if strings.EqualFold(tt.Name.Local, "style") {
					depth--
				}
			}
		}
		return nil, nil
	}

	// filter attributes
	var kept []xml.Attr
	for _, a := range start.Attr {
		val := a.Value
		// javascript: check
		noSpaceLower := strings.ToLower(val)
		// remove all whitespace
		noSpaceLower = strings.ReplaceAll(noSpaceLower, " ", "")
		noSpaceLower = strings.ReplaceAll(noSpaceLower, "\t", "")
		noSpaceLower = strings.ReplaceAll(noSpaceLower, "\n", "")
		noSpaceLower = strings.ReplaceAll(noSpaceLower, "\r", "")
		if strings.HasPrefix(noSpaceLower, "javascript:") {
			continue
		}
		// href handling
		if strings.EqualFold(a.Name.Local, "href") {
			trimmed := strings.TrimSpace(val)
			if !strings.HasPrefix(trimmed, "#") {
				continue
			}
			// keep href with original value but escaped later
			kept = append(kept, a)
			continue
		}
		// build lower name for whitelist check
		localLower := strings.ToLower(a.Name.Local)
		spaceLower := strings.ToLower(a.Name.Space)
		// map known url to prefix
		if spaceLower == "http://www.w3.org/1999/xlink" {
			spaceLower = "xlink"
		}
		fullLower := localLower
		if spaceLower != "" {
			fullLower = spaceLower + ":" + localLower
		}
		if isAllowedAttr(fullLower) || isAllowedAttr(localLower) {
			kept = append(kept, a)
		}
	}

	// now parse inner contents until matching EndElement
	var inner []content
	for {
		tok, err := dec.Token()
		if err != nil {
			return nil, fmt.Err("svg: el archivo no es un SVG valido")
		}
		switch t := tok.(type) {
		case xml.StartElement:
			child, err := sanitizeElement(dec, t)
			if err != nil {
				return nil, err
			}
			inner = append(inner, child...)
		case xml.EndElement:
			if strings.EqualFold(t.Name.Local, start.Name.Local) {
				if !isAllowedElement(low) {
					return inner, nil
				}
				n := &node{
					name:    start.Name.Local,
					attrs:   kept,
					content: inner,
				}
				return []content{{elem: n}}, nil
			}
			// mismatched end -> invalid xml
			return nil, fmt.Err("svg: el archivo no es un SVG valido")
		case xml.CharData:
			s := string(t)
			if strings.TrimSpace(s) == "" {
				continue
			}
			inner = append(inner, content{isText: true, text: s})
		case xml.Comment, xml.Directive, xml.ProcInst:
			continue
		}
	}
}

func writeNode(buf *bytes.Buffer, n *node) {
	buf.WriteString("<")
	buf.WriteString(n.name)
	for _, a := range n.attrs {
		name := attrName(a.Name)
		buf.WriteString(" ")
		buf.WriteString(name)
		buf.WriteString(`="`)
		buf.WriteString(escapeAttr(a.Value))
		buf.WriteString(`"`)
	}
	if len(n.content) == 0 {
		// for svg root always use closing tag for idempotence, not self-closing
		if n.name == "svg" {
			buf.WriteString("></")
			buf.WriteString(n.name)
			buf.WriteString(">")
			return
		}
		buf.WriteString("/>")
		return
	}
	buf.WriteString(">")
	for _, c := range n.content {
		if c.isText {
			buf.WriteString(escapeText(c.text))
		} else if c.elem != nil {
			writeNode(buf, c.elem)
		}
	}
	buf.WriteString("</")
	buf.WriteString(n.name)
	buf.WriteString(">")
}

func attrName(name xml.Name) string {
	local := name.Local
	space := name.Space
	if strings.ToLower(space) == "http://www.w3.org/1999/xlink" {
		space = "xlink"
	}
	if space != "" {
		if strings.ToLower(space) == "xmlns" {
			return space + ":" + local
		}
		return space + ":" + local
	}
	return local
}

func escapeAttr(s string) string {
	// must escape &, ", <, >
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

func escapeText(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}
