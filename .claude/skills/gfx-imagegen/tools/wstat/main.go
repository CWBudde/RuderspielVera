// wstat prints colour statistics (mean RGB, CIELAB L*, sigma L*, chroma, hue) of opaque pixels,
// alpha statistics, and optional per-region stats over a grid.
package main

import (
	"fmt"
	"image"
	"image/color"
	_ "image/png"
	"math"
	"os"
	"strconv"
)

func lin(c float64) float64 {
	c /= 255
	if c <= 0.04045 {
		return c / 12.92
	}
	return math.Pow((c+0.055)/1.055, 2.4)
}

func f(t float64) float64 {
	if t > 216.0/24389 {
		return math.Cbrt(t)
	}
	return (24389.0/27*t + 16) / 116
}

func lab(r, g, b float64) (float64, float64, float64) {
	R, G, B := lin(r), lin(g), lin(b)
	x := (0.4124*R + 0.3576*G + 0.1805*B) / 0.95047
	y := 0.2126*R + 0.7152*G + 0.0722*B
	z := (0.0193*R + 0.1192*G + 0.9505*B) / 1.08883
	fx, fy, fz := f(x), f(y), f(z)
	return 116*fy - 16, 500 * (fx - fy), 200 * (fy - fz)
}

type acc struct{ n, r, g, b, l, l2, a, bb float64 }

func (s *acc) add(r, g, b float64) {
	L, A, B := lab(r, g, b)
	s.n++
	s.r += r
	s.g += g
	s.b += b
	s.l += L
	s.l2 += L * L
	s.a += A
	s.bb += B
}

func (s acc) String() string {
	if s.n == 0 {
		return "no opaque pixels"
	}
	n := s.n
	ml := s.l / n
	sd := math.Sqrt(math.Max(0, s.l2/n-ml*ml))
	a, b := s.a/n, s.bb/n
	h := math.Atan2(b, a) * 180 / math.Pi
	if h < 0 {
		h += 360
	}
	return fmt.Sprintf("n=%.0f RGB=(%.0f,%.0f,%.0f) L*=%.1f sdL*=%.1f a*=%.1f b*=%.1f C=%.1f h=%.0f", n, s.r/n, s.g/n, s.b/n, ml, sd, a, b, math.Hypot(a, b), h)
}

func main() {
	fh, err := os.Open(os.Args[1])
	if err != nil {
		panic(err)
	}
	img, _, err := image.Decode(fh)
	if err != nil {
		panic(err)
	}
	gx, gy := 0, 0
	if len(os.Args) > 3 {
		gx, _ = strconv.Atoi(os.Args[2])
		gy, _ = strconv.Atoi(os.Args[3])
	}
	bd := img.Bounds()
	var all acc
	var a0, a255, amid float64
	cells := make([]acc, gx*gy)
	for y := bd.Min.Y; y < bd.Max.Y; y++ {
		for x := bd.Min.X; x < bd.Max.X; x++ {
			c := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
			r, g, b, a := uint32(c.R)<<8, uint32(c.G)<<8, uint32(c.B)<<8, uint32(c.A)*257
			switch {
			case a == 0:
				a0++
			case a >= 250*257:
				a255++
			default:
				amid++
			}
			if a < 250*257 {
				continue
			}
			R, G, B := float64(r>>8), float64(g>>8), float64(b>>8)
			all.add(R, G, B)
			if gx > 0 {
				cx := (x - bd.Min.X) * gx / bd.Dx()
				cy := (y - bd.Min.Y) * gy / bd.Dy()
				cells[cy*gx+cx].add(R, G, B)
			}
		}
	}
	t := float64(bd.Dx() * bd.Dy())
	fmt.Printf("%dx%d alpha0=%.1f%% opaque=%.1f%% partial=%.1f%%\n", bd.Dx(), bd.Dy(), 100*a0/t, 100*a255/t, 100*amid/t)
	fmt.Println("all:", all)
	for i, c := range cells {
		fmt.Printf("cell r%d c%d: %v\n", i/gx, i%gx, c)
	}
}
