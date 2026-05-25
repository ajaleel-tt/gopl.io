// Exercise 7.16: web-based calculator.
//
// Flow per request:
//   1. GET  /      -> render empty form
//   2. POST /      -> parse + check the expression. The set of variables
//                     is discovered from the AST. If the form already
//                     carries values for every variable, evaluate and
//                     render the result. Otherwise re-render the form
//                     with one input per discovered variable, preserving
//                     anything the user has already filled in.
//
// All eval errors (parse, check, bad number, divide-by-zero panic) land
// in the page's error slot — the server never crashes the request.
//
// Uses ex7.14's eval package so min(...) and max(...) work alongside
// pow / sin / sqrt.
package main

import (
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"gopl.io/ch7/ex7.14/eval"
)

var addr = flag.String("addr", ":8000", "HTTP listen address")

func main() {
	flag.Parse()
	http.HandleFunc("/", handle)
	log.Printf("listening on %s", *addr)
	log.Fatal(http.ListenAndServe(*addr, nil))
}

// page is the template data model.
type page struct {
	Expr     string         // raw user expression
	Pretty   string         // expr.String() once parsed
	Vars     []varInput     // discovered variables, with current values
	Result   string         // formatted result on success
	Error    string         // any error message
	Examples []string       // shown alongside the form for quick links
}

type varInput struct {
	Name  string
	Value string // raw string from the form, preserved on re-render
}

var examples = []string{
	"5 / 9 * (F - 32)",
	"sqrt(A / pi)",
	"pow(x, 3) + pow(y, 3)",
	"min(x, y, 3)",
	"max(-x, sqrt(x))",
}

func handle(w http.ResponseWriter, r *http.Request) {
	data := page{Examples: examples}

	// Both GET (?expr=...) and POST carry inputs through Form values.
	if err := r.ParseForm(); err != nil {
		data.Error = "bad form: " + err.Error()
		render(w, data)
		return
	}
	data.Expr = strings.TrimSpace(r.FormValue("expr"))

	if data.Expr == "" {
		render(w, data)
		return
	}

	expr, err := eval.Parse(data.Expr)
	if err != nil {
		data.Error = "parse error: " + err.Error()
		render(w, data)
		return
	}
	data.Pretty = expr.String()

	varsSet := map[eval.Var]bool{}
	if err := expr.Check(varsSet); err != nil {
		data.Error = "check error: " + err.Error()
		render(w, data)
		return
	}

	// Collect variable names in deterministic order; pull values from
	// the form (empty string if absent).
	names := make([]string, 0, len(varsSet))
	for v := range varsSet {
		names = append(names, string(v))
	}
	sort.Strings(names)

	env := eval.Env{}
	missing := false
	for _, n := range names {
		raw := strings.TrimSpace(r.FormValue("var_" + n))
		vi := varInput{Name: n, Value: raw}
		data.Vars = append(data.Vars, vi)
		if raw == "" {
			missing = true
			continue
		}
		f, perr := strconv.ParseFloat(raw, 64)
		if perr != nil {
			data.Error = fmt.Sprintf("%s: %v", n, perr)
			missing = true
			continue
		}
		env[eval.Var(n)] = f
	}

	if missing {
		// First evaluation of a fresh expression: ask the user for the
		// values we just discovered.
		if data.Error == "" {
			data.Error = "enter a value for each variable"
		}
		render(w, data)
		return
	}

	result, err := safeEval(expr, env)
	if err != nil {
		data.Error = "eval error: " + err.Error()
		render(w, data)
		return
	}
	data.Result = strconv.FormatFloat(result, 'g', -1, 64)
	render(w, data)
}

func safeEval(e eval.Expr, env eval.Env) (result float64, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
		}
	}()
	return e.Eval(env), nil
}

var tmpl = template.Must(template.New("calc").Parse(`<!doctype html>
<html>
<head>
<meta charset="utf-8">
<title>calc</title>
<style>
  body { font: 14px/1.4 system-ui, sans-serif; max-width: 38rem; margin: 2rem auto; padding: 0 1rem; }
  h1 { margin: 0 0 1rem; }
  form { display: grid; gap: 0.6rem; }
  input[type=text] { font: inherit; padding: 0.4rem 0.5rem; width: 100%; box-sizing: border-box; }
  button { padding: 0.4rem 0.9rem; font: inherit; cursor: pointer; }
  .vars { display: grid; grid-template-columns: max-content 1fr; gap: 0.4rem 0.8rem; align-items: center; }
  .vars label { font-family: ui-monospace, monospace; }
  .error { color: #b00; }
  .result { font-size: 1.4rem; margin: 1rem 0 0; }
  .pretty { color: #555; font-family: ui-monospace, monospace; }
  .examples a { font-family: ui-monospace, monospace; }
  hr { border: none; border-top: 1px solid #ddd; margin: 1.5rem 0; }
</style>
</head>
<body>
<h1>calc</h1>

<form method="POST" action="/">
  <label>expression
    <input type="text" name="expr" value="{{.Expr}}" autofocus required>
  </label>

  {{if .Vars}}
  <div class="vars">
    {{range .Vars}}
      <label for="var_{{.Name}}">{{.Name}} =</label>
      <input id="var_{{.Name}}" type="text" name="var_{{.Name}}" value="{{.Value}}">
    {{end}}
  </div>
  {{end}}

  <div><button type="submit">evaluate</button></div>
</form>

{{if .Error}}<p class="error">{{.Error}}</p>{{end}}

{{if ne .Result ""}}
  <p class="result">= {{.Result}}</p>
  {{if .Pretty}}<p class="pretty">{{.Pretty}}</p>{{end}}
{{end}}

<hr>
<p>examples:</p>
<ul class="examples">
{{range .Examples}}<li><a href="/?expr={{.}}">{{.}}</a></li>{{end}}
</ul>
</body>
</html>
`))

func render(w http.ResponseWriter, data page) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("template: %v", err)
	}
}
