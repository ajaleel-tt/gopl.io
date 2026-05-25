package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRun(t *testing.T) {
	cases := []struct {
		name        string
		input       string  // lines fed to stdin (expression then var values)
		wantOutSub  string  // substring expected in stdout
		wantErrSub  string  // substring expected in error (empty = no error)
	}{
		{
			name:       "happy path with vars",
			input:      "5 / 9 * (F - 32)\n212\n",
			wantOutSub: "= 100\n",
		},
		{
			name:       "no vars",
			input:      "1 + 2 * 3\n",
			wantOutSub: "= 7\n",
		},
		{
			name:       "min via ex7.14",
			input:      "min(x, y, 3)\n7\n5\n",
			wantOutSub: "= 3\n",
		},
		{
			name:       "re-prompts on bad number",
			input:      "x + 1\nnotanumber\n4\n",
			wantOutSub: "= 5\n",
		},
		{
			name:       "parse error reported",
			input:      "x % 2\n",
			wantErrSub: "parse:",
		},
		{
			name:       "check error reported",
			input:      "log(10)\n",
			wantErrSub: "check:",
		},
		{
			name:       "empty input rejected",
			input:      "\n",
			wantErrSub: "no expression",
		},
		{
			name:       "EOF while reading variable",
			input:      "x + 1\n", // no value supplied for x
			wantErrSub: "reading value for x",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var out bytes.Buffer
			err := run(strings.NewReader(c.input), &out)
			if c.wantErrSub == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v\noutput:\n%s", err, out.String())
				}
				if !strings.Contains(out.String(), c.wantOutSub) {
					t.Errorf("output missing %q\ngot:\n%s", c.wantOutSub, out.String())
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error containing %q, got nil; output:\n%s", c.wantErrSub, out.String())
			}
			if !strings.Contains(err.Error(), c.wantErrSub) {
				t.Errorf("error %q missing substring %q", err.Error(), c.wantErrSub)
			}
		})
	}
}
