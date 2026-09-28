// Command iconvars derives the in-app icon variants from the brand logo, so the
// picker always shows one artwork in three colours instead of unrelated images.
//
// Usage:
//
//	go run ./tools/iconvars -src assets/logo.png -out frontend/src/assets/icons
//
// The source must be a square PNG with a transparent background. Run `make icons`
// to regenerate the variants and the bundle icon together.
package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
)

// variant is one hue/lightness treatment of the source logo.
type variant struct {
	name  string
	hue   float64 // rotation in [0,1)
	sat   float64 // saturation multiplier
	light float64 // lightness multiplier
}

// variants must stay in sync with IconKey in frontend/src/lib/settings.ts.
var variants = []variant{
	{name: "default", hue: 0, sat: 1, light: 1},
	{name: "midnight", hue: 0.045, sat: 1.12, light: 0.92},
	{name: "citrus", hue: -0.455, sat: 1.05, light: 1.06},
}

func main() {
	src := flag.String("src", "assets/logo.png", "square PNG source with transparency")
	out := flag.String("out", "frontend/src/assets/icons", "directory the variants are written to")
	bundle := flag.String("bundle", "", "also write the 1024px opaque square app icon to this path")
	flag.Parse()

	if err := generate(*src, *out, *bundle); err != nil {
		fmt.Fprintln(os.Stderr, "iconvars:", err)
		os.Exit(1)
	}
}

func generate(src, out, bundle string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()

	base, err := png.Decode(f)
	if err != nil {
		return fmt.Errorf("%s must be a PNG: %w", src, err)
	}
	b := base.Bounds()
	if b.Dx() != b.Dy() {
		return fmt.Errorf("%s must be square, got %dx%d", src, b.Dx(), b.Dy())
	}
	if err := os.MkdirAll(out, 0o755); err != nil {
		return err
	}
	if bundle != "" {
		if err := writeBundle(base, bundle); err != nil {
			return fmt.Errorf("write %s: %w", bundle, err)
		}
		fmt.Printf("✔ %s (1024x1024, opaque square)\n", bundle)
	}
	// Variants fill their canvas too, so the runtime Dock icon matches the bundle.
	filled := fillCanvas(base, b.Dx(), 0)
	base, b = filled, filled.Bounds()

	for _, v := range variants {
		dst := filepath.Join(out, v.name+".png")
		if err := writeVariant(dst, base, b, v); err != nil {
			return fmt.Errorf("write %s: %w", dst, err)
		}
		fmt.Printf("✔ %s (%dx%d)\n", dst, b.Dx(), b.Dy())
	}
	return nil
}

func writeVariant(dst string, src image.Image, b image.Rectangle, v variant) error {
	out := image.NewRGBA(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, a := src.At(x, y).RGBA()
			rr, gg, bb := uint8(r>>8), uint8(g>>8), uint8(bl>>8)
			if v.hue != 0 || v.sat != 1 || v.light != 1 {
				h, s, l := rgbToHSL(rr, gg, bb)
				h = math.Mod(h+v.hue+1, 1)
				s = math.Min(1, s*v.sat)
				l = math.Max(0, math.Min(1, l*v.light))
				rr, gg, bb = hslToRGB(h, s, l)
			}
			// Alpha is copied untouched: the transparent corners are what make
			// the icon sit correctly in the Dock and in the settings cards.
			out.SetRGBA(x, y, color.RGBA{R: rr, G: gg, B: bb, A: uint8(a >> 8)})
		}
	}

	f, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, out)
}

// rgbToHSL and hslToRGB live here because image/color ships neither direction.

func rgbToHSL(r8, g8, b8 uint8) (float64, float64, float64) {
	r, g, b := float64(r8)/255, float64(g8)/255, float64(b8)/255
	max := math.Max(r, math.Max(g, b))
	min := math.Min(r, math.Min(g, b))
	l := (max + min) / 2
	if max == min {
		return 0, 0, l
	}
	d := max - min
	var h float64
	switch max {
	case r:
		h = (g - b) / d
		if g < b {
			h += 6
		}
	case g:
		h = (b-r)/d + 2
	default:
		h = (r-g)/d + 4
	}
	h /= 6
	s := d
	if l > 0.5 {
		s /= 2 - max - min
	} else {
		s /= max + min
	}
	return h, s, l
}

func hslToRGB(h, s, l float64) (uint8, uint8, uint8) {
	if s == 0 {
		v := uint8(math.Round(255 * l))
		return v, v, v
	}
	q := l * (1 + s)
	if l >= 0.5 {
		q = l + s - l*s
	}
	p := 2*l - q
	channel := func(t float64) uint8 {
		for t < 0 {
			t += 1
		}
		for t > 1 {
			t -= 1
		}
		switch {
		case t < 1.0/6:
			return uint8(math.Round(255 * (p + (q-p)*6*t)))
		case t < 1.0/2:
			return uint8(math.Round(255 * q))
		case t < 2.0/3:
			return uint8(math.Round(255 * (p + (q-p)*(2.0/3-t)*6)))
		default:
			return uint8(math.Round(255 * p))
		}
	}
	return channel(h + 1.0/3), channel(h), channel(h - 1.0/3)
}
