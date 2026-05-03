// Exercise 5.9: expand replaces each substring "$foo" in s with f("foo").
package main

import (
	"fmt"
	"regexp"
	"strings"
)

var varPattern = regexp.MustCompile(`\$(\w+)`)

func expand(s string, f func(string) string) string {
	return varPattern.ReplaceAllStringFunc(s, func(match string) string {
		return f(match[1:])
	})
}

func main() {
	template := "Hello $name, welcome to $place!"
	values := map[string]string{
		"name":  "Amizan",
		"place": "Go",
	}

	result := expand(template, func(key string) string {
		if v, ok := values[key]; ok {
			return v
		}
		return "$" + key
	})
	fmt.Println(result)

	upper := expand("$hello $world", strings.ToUpper)
	fmt.Println(upper)
}
