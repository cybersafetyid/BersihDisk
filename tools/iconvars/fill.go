package main

import (
	"image"
	"image/png"
	"math"
	"os"
)

// alphaCutoff ignores near-transparent pixels (anti-aliasing haze, stray
// semi-transparent margins) when measuring where the artwork really is.
const alphaCutoff = 8 << 8

// fillCanvas crops src to the bounding box of its visible pixels and scales that
// square to size x size. An icon drawn with a transparent margin otherwise shows
// a visible gap around it in the Dock, next to icons that run edge to edge.
// Alpha is preserved, so the rounded corners stay transparent.
func fillCanvas(src image.Image, size int) *image.NRGBA {
	b := src.Bounds()
	minX, minY, maxX, maxY := b.Max.X, b.Max.Y, b.Min.X, b.Min.Y
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if _, _, _, a := src.At(x, y).RGBA(); a > alphaCutoff {
				minX, maxX = min(minX, x), max(maxX, x+1)
				minY, maxY = min(minY, y), max(maxY, y+1)
			}
		}
	}
	if maxX <= minX || maxY <= minY {
		minX, minY, maxX, maxY = b.Min.X, b.Min.Y, b.Max.X, b.Max.Y
	}
	// Square crop around the artwork's centre.
	side := max(maxX-minX, maxY-minY)
	x0 := (minX + maxX - side) / 2
	y0 := (minY + maxY - side) / 2

	out := image.NewNRGBA(image.Rect(0, 0, size, size))
	scale := float64(side) / float64(size)
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			sx := (float64(x)+0.5)*scale - 0.5 + float64(x0)
			sy := (float64(y)+0.5)*scale - 0.5 + float64(y0)
			r, g, bl, a := bilinear(src, sx, sy)
			if a > 0 {
				i := out.PixOffset(x, y)
				out.Pix[i+0] = clamp8(r / a * 255)
				out.Pix[i+1] = clamp8(g / a * 255)
				out.Pix[i+2] = clamp8(bl / a * 255)
				out.Pix[i+3] = clamp8(a * 255)
			}
		}
	}
	return out
}

// bilinear samples src at a fractional position, in premultiplied [0,1] channels
// so transparent pixels do not bleed their colour into the edge.
func bilinear(src image.Image, x, y float64) (r, g, b, a float64) {
	bd := src.Bounds()
	x0, y0 := int(math.Floor(x)), int(math.Floor(y))
	fx, fy := x-float64(x0), y-float64(y0)
	for dy := 0; dy <= 1; dy++ {
		for dx := 0; dx <= 1; dx++ {
			px := min(max(x0+dx, bd.Min.X), bd.Max.X-1)
			py := min(max(y0+dy, bd.Min.Y), bd.Max.Y-1)
			w := (1 - math.Abs(float64(dx)-fx)) * (1 - math.Abs(float64(dy)-fy))
			cr, cg, cb, ca := src.At(px, py).RGBA()
			r += w * float64(cr) / 65535
			g += w * float64(cg) / 65535
			b += w * float64(cb) / 65535
			a += w * float64(ca) / 65535
		}
	}
	return
}

func clamp8(v float64) uint8 {
	return uint8(math.Max(0, math.Min(255, math.Round(v))))
}

// writeBundle writes the edge-to-edge app icon (1024px, what macOS/Windows/Linux
// bundles are built from).
func writeBundle(src image.Image, dst string) error {
	f, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, fillCanvas(src, 1024))
}
