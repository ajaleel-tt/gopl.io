// Exercise 7.15: read a single arithmetic expression from stdin, prompt
// for the value of each variable it references, then evaluate it.
//
// All error paths are handled without panicking:
//   - parse errors    -> reported and exit 1
//   - check errors    -> reported and exit 1 (unknown function, bad arity)
//   - bad number      -> re-prompt
//   - EOF on prompt   -> reported and exit 1
//   - eval panics     -> recovered and reported (e.g. unsupported operator)
//
// The eval package is the ex7.14 variant, so min(...) and max(...) work
// in addition to pow / sin / sqrt.
package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"

	"gopl.io/ch7/ex7.14/eval"
)

func main() {
	if err := run(os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(in io.Reader, out io.Writer) error {
	r := bufio.NewReader(in)

	fmt.Fprint(out, "expression> ")
	line, err := readLine(r)
	if err != nil {
		return fmt.Errorf("reading expression: %w", err)
	}
	if strings.TrimSpace(line) == "" {
		return fmt.Errorf("no expression provided")
	}

	expr, err := eval.Parse(line)
	if err != nil {
		return fmt.Errorf("parse: %w", err)
	}

	vars := map[eval.Var]bool{}
	if err := expr.Check(vars); err != nil {
		return fmt.Errorf("check: %w", err)
	}

	env, err := promptForVars(r, out, vars)
	if err != nil {
		return err
	}

	result, err := safeEval(expr, env)
	if err != nil {
		return fmt.Errorf("eval: %w", err)
	}

	fmt.Fprintf(out, "= %g\n", result)
	return nil
}

// promptForVars asks the user for a value for each variable. Bad input
// re-prompts; EOF aborts.
func promptForVars(r *bufio.Reader, out io.Writer, vars map[eval.Var]bool) (eval.Env, error) {
	names := make([]eval.Var, 0, len(vars))
	for v := range vars {
		names = append(names, v)
	}
	sort.Slice(names, func(i, j int) bool { return names[i] < names[j] })

	env := eval.Env{}
	for _, name := range names {
		for {
			fmt.Fprintf(out, "  %s = ", name)
			line, err := readLine(r)
			if err != nil {
				return nil, fmt.Errorf("reading value for %s: %w", name, err)
			}
			line = strings.TrimSpace(line)
			f, perr := strconv.ParseFloat(line, 64)
			if perr != nil {
				fmt.Fprintf(out, "    not a number (%v); try again\n", perr)
				continue
			}
			env[name] = f
			break
		}
	}
	return env, nil
}

// safeEval catches panics from Eval (e.g. unsupported operator) and
// converts them into errors.
func safeEval(e eval.Expr, env eval.Env) (result float64, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
		}
	}()
	return e.Eval(env), nil
}

// readLine returns the next line without the trailing newline. Returns
// io.EOF if the reader is exhausted without producing any data.
func readLine(r *bufio.Reader) (string, error) {
	line, err := r.ReadString('\n')
	if err == io.EOF && line == "" {
		return "", io.EOF
	}
	if err != nil && err != io.EOF {
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}
