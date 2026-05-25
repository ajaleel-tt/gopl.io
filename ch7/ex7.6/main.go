// Exercise 7.6: tempflag extended to accept Kelvin input.
//
// Usage: ex7.6 -temp 100C    → 100°C
//
//	ex7.6 -temp 212F    → 100°C (converted from Fahrenheit)
//	ex7.6 -temp 300K    → 26.85°C (converted from Kelvin)
//	ex7.6 -temp 300°K   → same as 300K
//
// Internal storage is always Celsius; the unit suffix only controls how the
// input string is interpreted.
package main

import (
	"flag"
	"fmt"
)

type Celsius float64
type Fahrenheit float64
type Kelvin float64

func CToF(c Celsius) Fahrenheit { return Fahrenheit(c*9.0/5.0 + 32.0) }
func FToC(f Fahrenheit) Celsius { return Celsius((f - 32.0) * 5.0 / 9.0) }
func CToK(c Celsius) Kelvin     { return Kelvin(c + 273.15) }
func KToC(k Kelvin) Celsius     { return Celsius(k - 273.15) }

func (c Celsius) String() string    { return fmt.Sprintf("%g°C", c) }
func (f Fahrenheit) String() string { return fmt.Sprintf("%g°F", f) }
func (k Kelvin) String() string     { return fmt.Sprintf("%g°K", k) }

// celsiusFlag satisfies flag.Value. It stores the temperature internally as
// Celsius regardless of the unit the user supplied.
type celsiusFlag struct{ Celsius }

func (f *celsiusFlag) Set(s string) error {
	var unit string
	var value float64
	fmt.Sscanf(s, "%f%s", &value, &unit)
	switch unit {
	case "C", "°C":
		f.Celsius = Celsius(value)
		return nil
	case "F", "°F":
		f.Celsius = FToC(Fahrenheit(value))
		return nil
	case "K", "°K":
		f.Celsius = KToC(Kelvin(value))
		return nil
	}
	return fmt.Errorf("invalid temperature %q", s)
}

// CelsiusFlag defines a temperature flag that accepts C, F, or K input
// and returns the value normalized to Celsius.
func CelsiusFlag(name string, value Celsius, usage string) *Celsius {
	f := celsiusFlag{value}
	flag.CommandLine.Var(&f, name, usage)
	return &f.Celsius
}

var temp = CelsiusFlag("temp", 20.0, "the temperature (e.g., 100C, 212F, 300K)")

func main() {
	flag.Parse()
	fmt.Println(*temp)
}
