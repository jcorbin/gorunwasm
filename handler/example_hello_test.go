package handler_test

// +build js

import "syscall/js"

var (
	window   = js.Global().Get("window")
	document = js.Global().Get("document")
)

func Example_hello() {
	window.Call("addEventListener", "DOMContentLoaded",
		js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			document.Get("body").Set("innerText", "hello world")
		}))
}
