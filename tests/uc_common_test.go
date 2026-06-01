//go:build wasm

package svg_test

import (
	"syscall/js"
	"testing"

	"github.com/tinywasm/dom"
)

func SetupDOM(t *testing.T) js.Value {
	doc := js.Global().Get("document")
	body := doc.Get("body")

	root := doc.Call("getElementById", "root")
	if root.IsNull() {
		root = doc.Call("createElement", "div")
		root.Set("id", "root")
		body.Call("appendChild", root)
	} else {
		root.Set("innerHTML", "")
	}

	dom.SetLog(func(v ...any) {
		t.Log(v...)
	})

	return doc
}

type TestReference struct {
	val js.Value
}

func (r *TestReference) GetAttr(key string) string {
	val := r.val.Call("getAttribute", key)
	if val.IsNull() {
		return ""
	}
	return val.String()
}

func (r *TestReference) Value() string        { return r.val.Get("value").String() }
func (r *TestReference) SetValue(value string) { r.val.Set("value", value) }
func (r *TestReference) SetAttr(key, value string) { r.val.Call("setAttribute", key, value) }
func (r *TestReference) RemoveAttr(key string)      { r.val.Call("removeAttribute", key) }
func (r *TestReference) SetText(text string)        { r.val.Set("textContent", text) }
func (r *TestReference) Checked() bool              { return r.val.Get("checked").Bool() }
func (r *TestReference) Focus()                     { r.val.Call("focus") }

func (r *TestReference) On(eventType string, handler func(event dom.Event)) {
	fn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		var e dom.Event
		if len(args) > 0 {
			e = &MockEvent{val: args[0]}
		}
		handler(e)
		return nil
	})
	r.val.Call("addEventListener", eventType, fn)
}

type MockEvent struct {
	val js.Value
}

func (e *MockEvent) PreventDefault()  { e.val.Call("preventDefault") }
func (e *MockEvent) StopPropagation() { e.val.Call("stopPropagation") }

func (e *MockEvent) TargetID() string {
	t := e.val.Get("target")
	if t.IsNull() || t.IsUndefined() {
		return ""
	}
	return t.Get("id").String()
}

func (e *MockEvent) TargetValue() string {
	t := e.val.Get("target")
	if t.IsNull() || t.IsUndefined() {
		return ""
	}
	return t.Get("value").String()
}

func (e *MockEvent) TargetChecked() bool {
	t := e.val.Get("target")
	if t.IsNull() || t.IsUndefined() {
		return false
	}
	return t.Get("checked").Bool()
}

func TriggerEvent(id, eventType, value string) {
	doc := js.Global().Get("document")
	rawEl := doc.Call("getElementById", id)
	if !rawEl.IsNull() && !rawEl.IsUndefined() {
		if value != "" {
			rawEl.Set("value", value)
		}
		event := js.Global().Get("Event").New(eventType, map[string]interface{}{"bubbles": true})
		rawEl.Call("dispatchEvent", event)
	}
}

func GetRef(id string) (*TestReference, bool) {
	doc := js.Global().Get("document")
	var val js.Value
	switch id {
	case "body":
		val = doc.Get("body")
	case "head":
		val = doc.Get("head")
	default:
		val = doc.Call("getElementById", id)
	}
	if val.IsNull() || val.IsUndefined() {
		return nil, false
	}
	return &TestReference{val: val}, true
}
