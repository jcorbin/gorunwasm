// +build !dev

package main

import (
	"net/http"
	"strings"
	"time"
)

var staticContentModTime = time.Now()

func staticHandler(name, content string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		http.ServeContent(w, req, name, staticContentModTime, strings.NewReader(content))
	})
}

func init() {
	indexHandler = staticHandler("index.html", "<!doctype html>\n\n<meta charset=\"utf-8\">\n<title>Go Run</title>\n\n<body>\n\n\t<footer id=\"status\">Loading...</footer>\n\t<script src=\"wasm_exec.js\"></script>\n\t<script src=\"index.js\" data-status-selector=\"#status\"></script>\n\n</body>\n")
	mux.Handle("/index.js", staticHandler("index.js", "global.GoRunner = class {\n\t// parseConfigData from an element's data-* attributes:\n\t// - data-href optional URL to fetch the WASM binary, defaults to main.wasm.\n\t// - data-status-selector may provide a dom query selector for a displaying\n\t//   build errors and (re)running the Go main.\n\t// - data-args may provide a JSON-encoded argument array to pass to the Go program.\n\t// - data-argv0 overrides the program name (argv0) that the Go program is\n\t//   invoked under; defaults to \"package_name.wasm\"\n\t// - any other data-* keys are passed as environment variables to the Go program.\n\tstatic parseConfigData(el) {\n\t\tconst cfg = {\n\t\t\tel: null,\n\t\t\thref: 'main.wasm',\n\t\t\targv0: null,\n\t\t\targs: null,\n\t\t\tenv: {},\n\t\t};\n\t\tfor (let i = 0; i < el.attributes.length; i++) {\n\t\t\tconst {nodeName, nodeValue} = el.attributes[i];\n\t\t\tconst dataMatch = /^data-(.+)/.exec(nodeName);\n\t\t\tif (!dataMatch) continue;\n\t\t\tconst name = dataMatch[1];\n\t\t\tswitch (name) {\n\t\t\t\tcase 'href':\n\t\t\t\t\tcfg.href = nodeValue;\n\t\t\t\t\tbreak;\n\t\t\t\tcase 'status-selector':\n\t\t\t\t\tcfg.el = document.querySelector(nodeValue);\n\t\t\t\t\tbreak;\n\t\t\t\tcase 'argv0':\n\t\t\t\t\tcfg.argv0 = nodeValue;\n\t\t\t\t\tbreak;\n\t\t\t\tcase 'args':\n\t\t\t\t\tcfg.args = JSON.parse(nodeValue);\n\t\t\t\t\tif (!Array.isArray(cfg.args)) throw new Error('data-args must be an array');\n\t\t\t\t\tbreak;\n\t\t\t\tdefault:\n\t\t\t\t\tcfg.env[name] = nodeValue;\n\t\t\t\t\tbreak;\n\t\t\t}\n\t\t}\n\t\treturn cfg;\n\t}\n\n\tconstructor(cfg) {\n\t\tthis.el = cfg.el;\n\t\tthis.href = cfg.href;\n\t\tthis.args = cfg.args;\n\t\tthis.env = cfg.env;\n\t\tthis.argv0 = cfg.argv0;\n\t\tthis.module = null;\n\t\tthis.load();\n\t}\n\n\tasync load() {\n\t\tconst parseContentType = (resp) => {\n\t\t\tconst match = /^([^;]+)/.exec(resp.headers.get('Content-Type'));\n\t\t\treturn match ? match[1] : '';\n\t\t};\n\n\t\tconst setTitle = (title) => {\n\t\t\tif (document.title === 'Go Run') {\n\t\t\t\tdocument.title += ': ' + title;\n\t\t\t}\n\t\t};\n\n\t\tlet resp = await fetch(this.href);\n\t\tif (parseContentType(resp) === 'application/json') {\n\t\t\tconst data = await resp.json();\n\t\t\tsetTitle(data.Package.ImportPath);\n\t\t\tif (this.el) {\n\t\t\t\tthis.el.innerHTML = `Building <tt>${data.Package.ImportPath}</tt>...`;\n\t\t\t}\n\t\t\tconst basename = data.Package.Dir.split('/').pop();\n\t\t\tif (!this.argv0) {\n\t\t\t\tthis.argv0 = basename + '.wasm';\n\t\t\t}\n\t\t\tresp = await fetch(data.Bin);\n\t\t} else {\n\t\t\tif (!this.argv0) {\n\t\t\t\tconst match = /\\/([^\\/]+$)/.exec(this.href);\n\t\t\t\tthis.argv0 = match ? match[1] : this.href;\n\t\t\t}\n\t\t\tsetTitle(this.argv0);\n\t\t}\n\n\t\tif (parseContentType(resp) === 'text/plain') { // TODO support text/html formatted error\n\t\t\tconst el = this.el || document.body;\n\t\t\tel.innerHTML = `<pre class=\"buildLog\"></pre>`;\n\t\t\tel.querySelector('pre').innerText = await resp.text();\n\t\t\treturn;\n\t\t}\n\n\t\tthis.module = await WebAssembly.compileStreaming(resp);\n\n\t\tif (this.el && !this.args) {\n\t\t\treturn this.interact();\n\t\t}\n\n\t\tlet argv = [this.argv0];\n\t\tif (this.args) {\n\t\t\targv = argv.concat(this.args);\n\t\t}\n\n\t\tif (this.el) {\n\t\t\tthis.el.innerHTML = 'Running...';\n\t\t\tthis.el.style.display = 'none';\n\t\t}\n\n\t\tawait this.run(argv);\n\n\t\tif (this.el) {\n\t\t\tthis.el.style.display = '';\n\t\t\tthis.el.innerHTML = 'Done.';\n\t\t}\n\t}\n\n\tasync interact() {\n\t\tthis.el.innerHTML = `<input class=\"argv\" size=\"40\" title=\"JSON-encoded ARGV\" /><button class=\"run\">Run</button>`;\n\t\tconst runButton = this.el.querySelector('button.run');\n\t\tconst argvInput = this.el.querySelector('input.argv');\n\n\t\targvInput.value = JSON.stringify([this.argv0]);\n\n\t\trunButton.onclick = async () => {\n\t\t\tif (runButton.disabled) return;\n\n\t\t\tconst argv = JSON.parse(argvInput.value);\n\n\t\t\trunButton.disabled = true;\n\t\t\trunButton.innerText = 'Running...';\n\t\t\tthis.el.style.display = 'none';\n\n\t\t\tconsole.clear();\n\t\t\tawait this.run(argv);\n\n\t\t\tthis.el.style.display = '';\n\t\t\trunButton.innerText = 'Run';\n\t\t\trunButton.disabled = false;\n\t\t};\n\t}\n\n\tasync run(argv) {\n\t\tconst go = new Go();\n\t\tgo.env = this.env;\n\t\tgo.argv = argv;\n\t\tconst instance = await WebAssembly.instantiate(this.module, go.importObject);\n\t\tawait go.run(instance);\n\t}\n};\n\nglobal.goRun = (() => {\n\tconst cfg = GoRunner.parseConfigData(document.currentScript);\n\treturn new GoRunner(cfg);\n})();\n"))
}
