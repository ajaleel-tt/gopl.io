package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Supported units:
//   kg  - kilograms       -> pounds (lb)
//   lb  - pounds          -> kilograms (kg)
//   m   - meters          -> feet (ft)
//   ft  - feet            -> meters (m)
//
// Input format: "<number><unit>", e.g. "70kg", "150lb", "1.8m", "6ft".
// Whitespace is allowed around the token, but not inside the number+unit.
//
// Examples:
//   go run main.go 70kg 180lb 6ft 2m
//   echo "70kg 6ft" | go run main.go
//   echo "70kg\n6ft\n" | go run main.go

func main() {
	var tokens []string

	if len(os.Args) > 1 {
		// Use command-line arguments as tokens
		tokens = os.Args[1:]
	} else {
		// Read from standard input
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Split(bufio.ScanWords)
		for scanner.Scan() {
			tokens = append(tokens, scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			fmt.Fprintf(os.Stderr, "error reading stdin: %v\n", err)
			os.Exit(1)
		}
	}

	if len(tokens) == 0 {
		fmt.Fprintln(os.Stderr, "no input provided; provide values like 70kg, 150lb, 6ft, 1.8m")
		os.Exit(1)
	}

	for _, tok := range tokens {
		if tok == "" {
			continue
		}
		convertToken(tok)
	}
}

func convertToken(tok string) {
	tok = strings.TrimSpace(tok)
	if tok == "" {
		return
	}

	valueStr, unit := splitNumberAndUnit(tok)
	if valueStr == "" || unit == "" {
		fmt.Fprintf(os.Stderr, "invalid input %q (expected something like 70kg or 6ft)\n", tok)
		return
	}

	value, err := strconv.ParseFloat(valueStr, 64)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid number in %q: %v\n", tok, err)
		return
	}

	switch strings.ToLower(unit) {
	case "kg":
		lb := kgToLb(value)
		fmt.Printf("%g kg = %.4g lb\n", value, lb)
	case "lb", "lbs":
		kg := lbToKg(value)
		fmt.Printf("%g lb = %.4g kg\n", value, kg)
	case "m", "meter", "meters":
		ft := mToFt(value)
		fmt.Printf("%g m = %.4g ft\n", value, ft)
	case "ft", "foot", "feet":
		m := ftToM(value)
		fmt.Printf("%g ft = %.4g m\n", value, m)
	default:
		fmt.Fprintf(os.Stderr, "unknown unit in %q (supported: kg, lb, m, ft)\n", tok)
	}
}

// splitNumberAndUnit splits something like "70kg" into ("70", "kg").
func splitNumberAndUnit(s string) (numberPart, unitPart string) {
	i := 0
	for i < len(s) && (s[i] == '+' || s[i] == '-' || s[i] == '.' || (s[i] >= '0' && s[i] <= '9')) {
		i++
	}
	if i == 0 || i == len(s) {
		return "", ""
	}
	return s[:i], s[i:]
}

// Conversion functions:

func kgToLb(kg float64) float64 {
	// 1 kg ≈ 2.2046226218 lb
	return kg * 2.2046226218
}

func lbToKg(lb float64) float64 {
	// 1 lb ≈ 0.45359237 kg
	return lb * 0.45359237
}

func mToFt(m float64) float64 {
	// 1 m ≈ 3.280839895 ft
	return m * 3.280839895
}

func ftToM(ft float64) float64 {
	// 1 ft ≈ 0.3048 m
	return ft * 0.3048
}
