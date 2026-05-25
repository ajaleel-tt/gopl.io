package main

import (
	"math"
	"testing"
)

func approxEqual(a, b Celsius) bool {
	return math.Abs(float64(a-b)) < 1e-9
}

func TestSetCelsius(t *testing.T) {
	cases := []struct {
		input string
		want  Celsius
	}{
		{"0C", 0},
		{"100C", 100},
		{"-40C", -40},
		{"20°C", 20},
		{"3.14C", 3.14},
	}
	for _, tc := range cases {
		var f celsiusFlag
		if err := f.Set(tc.input); err != nil {
			t.Errorf("Set(%q): unexpected error: %v", tc.input, err)
			continue
		}
		if !approxEqual(f.Celsius, tc.want) {
			t.Errorf("Set(%q): got %v, want %v", tc.input, f.Celsius, tc.want)
		}
	}
}

func TestSetFahrenheit(t *testing.T) {
	cases := []struct {
		input string
		want  Celsius
	}{
		{"32F", 0},
		{"212F", 100},
		{"-40F", -40}, // -40 is the same in C and F
		{"32°F", 0},
	}
	for _, tc := range cases {
		var f celsiusFlag
		if err := f.Set(tc.input); err != nil {
			t.Errorf("Set(%q): unexpected error: %v", tc.input, err)
			continue
		}
		if !approxEqual(f.Celsius, tc.want) {
			t.Errorf("Set(%q): got %v, want %v", tc.input, f.Celsius, tc.want)
		}
	}
}

func TestSetKelvin(t *testing.T) {
	cases := []struct {
		input string
		want  Celsius
	}{
		{"0K", -273.15},
		{"273.15K", 0},
		{"373.15K", 100},
		{"300°K", 26.85},
		{"100K", -173.15},
	}
	for _, tc := range cases {
		var f celsiusFlag
		if err := f.Set(tc.input); err != nil {
			t.Errorf("Set(%q): unexpected error: %v", tc.input, err)
			continue
		}
		if !approxEqual(f.Celsius, tc.want) {
			t.Errorf("Set(%q): got %v, want %v", tc.input, f.Celsius, tc.want)
		}
	}
}

func TestSetInvalidUnit(t *testing.T) {
	cases := []string{"100", "100X", "100R", "abc", "100c"} // lowercase c is rejected
	for _, in := range cases {
		var f celsiusFlag
		err := f.Set(in)
		if err == nil {
			t.Errorf("Set(%q) returned nil error, want error", in)
		}
	}
}

func TestStringFormat(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"0C", "0°C"},
		{"100C", "100°C"},
		{"212F", "100°C"},
		{"273.15K", "0°C"},
	}
	for _, tc := range cases {
		var f celsiusFlag
		if err := f.Set(tc.input); err != nil {
			t.Errorf("Set(%q): %v", tc.input, err)
			continue
		}
		if got := f.Celsius.String(); got != tc.want {
			t.Errorf("Set(%q).String() = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestConversionRoundTrip(t *testing.T) {
	// 100°C → 212°F → 100°C
	c := Celsius(100)
	if got := FToC(CToF(c)); !approxEqual(got, c) {
		t.Errorf("C→F→C round trip lost precision: %v → %v", c, got)
	}
	// 0°C → 273.15K → 0°C
	if got := KToC(CToK(c)); !approxEqual(got, c) {
		t.Errorf("C→K→C round trip lost precision: %v → %v", c, got)
	}
}
