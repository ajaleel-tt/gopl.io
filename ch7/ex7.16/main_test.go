package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func do(form url.Values) *httptest.ResponseRecorder {
	var req *http.Request
	if form == nil {
		req = httptest.NewRequest(http.MethodGet, "/", nil)
	} else {
		req = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	rr := httptest.NewRecorder()
	handle(rr, req)
	return rr
}

func TestHandle(t *testing.T) {
	cases := []struct {
		name    string
		form    url.Values // nil => GET /
		mustHas []string   // substrings the response must contain
		mustNot []string   // substrings it must NOT contain
	}{
		{
			name:    "GET empty",
			form:    nil,
			mustHas: []string{"<form", "expression"},
			// mustNot avoids strings that appear in CSS (".error", etc.).
			mustNot: []string{`class="result"`, `class="error">`},
		},
		{
			name: "expression with no vars evaluates",
			form: url.Values{"expr": {"1 + 2 * 3"}},
			mustHas: []string{
				"= 7", // result
				// html/template escapes '+' inside attribute/text contexts.
				"(1 &#43; (2 * 3))",
			},
			mustNot: []string{`class="error">`},
		},
		{
			name: "expression with vars first prompts",
			form: url.Values{"expr": {"5 / 9 * (F - 32)"}},
			mustHas: []string{
				`name="var_F"`,
				"enter a value",
			},
			mustNot: []string{"= "},
		},
		{
			name: "expression with vars evaluates when filled",
			form: url.Values{"expr": {"5 / 9 * (F - 32)"}, "var_F": {"212"}},
			mustHas: []string{"= 100"},
			mustNot: []string{"enter a value"},
		},
		{
			name:    "parse error rendered",
			form:    url.Values{"expr": {"x % 2"}},
			mustHas: []string{"parse error"},
			mustNot: []string{"= "},
		},
		{
			name:    "check error rendered",
			form:    url.Values{"expr": {"log(10)"}},
			mustHas: []string{"check error"},
			mustNot: []string{"= "},
		},
		{
			name:    "bad number rendered, vars still shown",
			form:    url.Values{"expr": {"x + 1"}, "var_x": {"notanumber"}},
			mustHas: []string{"x:", `name="var_x"`, "notanumber"},
			mustNot: []string{"= "},
		},
		{
			name:    "ex7.14 min via web",
			form:    url.Values{"expr": {"min(x, y, 3)"}, "var_x": {"7"}, "var_y": {"5"}},
			mustHas: []string{"= 3"},
		},
		{
			name:    "HTML escaping of user input",
			form:    url.Values{"expr": {"<script>alert(1)</script>"}},
			mustHas: []string{"&lt;script&gt;"},
			mustNot: []string{"<script>alert"},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rr := do(c.form)
			if rr.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", rr.Code)
			}
			body := rr.Body.String()
			for _, s := range c.mustHas {
				if !strings.Contains(body, s) {
					t.Errorf("missing %q in body:\n%s", s, body)
				}
			}
			for _, s := range c.mustNot {
				if strings.Contains(body, s) {
					t.Errorf("unexpected %q in body:\n%s", s, body)
				}
			}
		})
	}
}
