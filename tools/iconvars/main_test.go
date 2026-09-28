package main

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
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

// A logo drawn with a transparent margin must come out edge to edge, with the
// rounded corner still transparent.
func TestFillCanvasRemovesMargin(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 100, 100))
	for y := 20; y < 80; y++ {
		for x := 20; x < 80; x++ {
			src.SetNRGBA(x, y, color.NRGBA{R: 200, G: 10, B: 10, A: 255})
		}
	}
	out := fillCanvas(src, 64, 0)
	if got := out.NRGBAAt(1, 32); got.A != 255 {
		t.Errorf("left edge alpha = %d, want opaque artwork right at the edge", got.A)
	}
	if got := out.NRGBAAt(32, 62); got.R != 200 || got.A != 255 {
		t.Errorf("bottom edge = %+v, want the artwork colour", got)
	}
}

// The bundle icon is an opaque square: no transparent corner may survive.
func TestWriteBundleIsOpaqueSquare(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 100, 100))
	for y := 10; y < 90; y++ {
		for x := 10; x < 90; x++ {
			// A rounded tile: cut the corners.
			dx, dy := float64(min(x-10, 89-x)), float64(min(y-10, 89-y))
			if dx < 20 && dy < 20 && (20-dx)*(20-dx)+(20-dy)*(20-dy) > 400 {
				continue
			}
			src.SetNRGBA(x, y, color.NRGBA{R: 30, G: 40, B: 50, A: 255})
		}
	}
	dst := filepath.Join(t.TempDir(), "icon.png")
	if err := writeBundle(src, dst); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(dst)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	out, err := png.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	w := out.Bounds().Dx()
	for _, p := range [][2]int{{0, 0}, {w - 1, 0}, {0, w - 1}, {w - 1, w - 1}} {
		r, _, _, a := out.At(p[0], p[1]).RGBA()
		if a>>8 != 255 || r>>8 > 100 {
			t.Errorf("corner %v = r%d a%d, want an opaque dark pixel (no white plate colour)", p, r>>8, a>>8)
		}
	}
}
