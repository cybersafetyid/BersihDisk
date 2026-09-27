package main

import (
	"image/color"
	"testing"
)

// The first version of this tool silently produced black icons because the HSL
// conversion forgot the 255 scale, so the round trip is pinned here.
func TestHSLRoundTrip(t *testing.T) {
	cases := []color.RGBA{
		{79, 140, 255, 255},
		{0, 0, 0, 255},
		{255, 255, 255, 255},
		{12, 14, 18, 255},
		{180, 220, 250, 255},
		{255, 176, 32, 128},
	}
	for _, c := range cases {
		h, s, l := rgbToHSL(c.R, c.G, c.B)
		r, g, b := hslToRGB(h, s, l)
		if !near(c.R, r) || !near(c.G, g) || !near(c.B, b) {
			t.Errorf("round trip for %v gave %v, %v, %v", c, r, g, b)
		}
	}
}

// near allows the ±1 rounding that fixed-point conversion implies.
func near(a, b uint8) bool {
	if a > b {
		return a-b <= 1
	}
	return b-a <= 1
}

// The frontend picker is keyed by these names; renaming one must rename the other.
func TestVariantNamesMatchFrontendIconKey(t *testing.T) {
	want := []string{"default", "midnight", "citrus"}
	if len(variants) != len(want) {
		t.Fatalf("got %d variants, want %d", len(variants), len(want))
	}
	for i, v := range variants {
		if v.name != want[i] {
			t.Errorf("variants[%d].name = %q, want %q (keep IconKey in settings.ts in sync)", i, v.name, want[i])
		}
	}
}
