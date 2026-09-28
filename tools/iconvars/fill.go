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

// squareInset is how much of the artwork's side is trimmed on each edge to turn a
// rounded tile into a full square: the tile's corner radius is about 24% of its
// side, and its arc clears a square inset by radius x (1 - 1/sqrt 2), about 7%.
const squareInset = 0.075

// fillCanvas crops src to the bounding box of its visible pixels, trims inset (a
// fraction of the side) off every edge, and scales the result to size x size.
// An icon drawn with a transparent margin otherwise shows a visible gap around it
// in the Dock, next to icons that run edge to edge. With inset 0 the alpha is
// preserved, so rounded corners stay transparent.
func fillCanvas(src image.Image, size int, inset float64) *image.NRGBA {
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
	cx, cy := float64(minX+maxX)/2, float64(minY+maxY)/2
	crop := float64(side) * (1 - 2*inset)
	x0 := cx - crop/2
	y0 := cy - crop/2

	out := image.NewNRGBA(image.Rect(0, 0, size, size))
	scale := crop / float64(size)
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			sx := (float64(x)+0.5)*scale - 0.5 + x0
			sy := (float64(y)+0.5)*scale - 0.5 + y0
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

// writeBundle writes the app icon the bundles are built from: a fully opaque
// 1024px square with no rounded corners. macOS applies its own rounded mask; an
// icon with transparent corners is treated as a legacy icon and shown shrunk on a
// white plate, which is exactly the white rim this avoids.
func writeBundle(src image.Image, dst string) error {
	icon := fillCanvas(src, 1024, squareInset)
	for i := 3; i < len(icon.Pix); i += 4 {
		icon.Pix[i] = 255
	}
	f, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, icon)
}
