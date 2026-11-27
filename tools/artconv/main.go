// Command artconv shrinks the game art in web/src/assets/art (top level only; people/ is never
// touched). Opaque PNGs become quality JPEGs (the PNG is removed), images with transparency stay
// PNG. Images wider than -max are downscaled. JPEGs are re-encoded only when that makes them
// smaller. WebP would be smaller still, but Go has no WebP encoder and cwebp is not required.
//
//	go run ./tools/artconv -dir web/src/assets/art -q 82 -max 900
package main

import (
	"bytes"
	"flag"
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"strings"

	xdraw "golang.org/x/image/draw"
)

func main() {
	dir := flag.String("dir", "web/src/assets/art", "art directory (non-recursive)")
	quality := flag.Int("q", 82, "JPEG quality")
	maxW := flag.Int("max", 900, "maximum width in pixels (0 = keep)")
	dry := flag.Bool("n", false, "dry run: report only")

	flag.Parse()

	entries, err := os.ReadDir(*dir)
	if err != nil {
		log.Fatal(err)
	}
	var before, after int64

	for _, e := range entries {
		if e.IsDir() {
			continue // people/ and anything else nested stays untouched
		}

		ext := strings.ToLower(filepath.Ext(e.Name()))
		if ext != ".png" && ext != ".jpg" && ext != ".jpeg" {
			continue
		}

		path := filepath.Join(*dir, e.Name())

		b, a, err := convert(path, ext, *quality, *maxW, *dry)
		if err != nil {
			log.Fatalf("%s: %v", path, err)
		}

		before += b
		after += a
	}

	fmt.Printf("total %.2f MB -> %.2f MB\n", mb(before), mb(after))
}

func convert(path, ext string, quality, maxW int, dry bool) (int64, int64, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return 0, 0, err
	}

	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return 0, 0, err
	}

	img = downscale(img, maxW)
	out, target := new(bytes.Buffer), path

	if ext == ".png" && !opaque(img) {
		enc := png.Encoder{CompressionLevel: png.BestCompression}

		err := enc.Encode(out, img)
		if err != nil {
			return 0, 0, err
		}
	} else {
		err := jpeg.Encode(out, img, &jpeg.Options{Quality: quality})
		if err != nil {
			return 0, 0, err
		}

		target = strings.TrimSuffix(path, filepath.Ext(path)) + ".jpg"
	}

	if target == path && int64(out.Len()) >= int64(len(raw)) {
		fmt.Printf("%-40s %7.0f kB (kept)\n", filepath.Base(path), kb(len(raw)))
		return int64(len(raw)), int64(len(raw)), nil
	}

	fmt.Printf("%-40s %7.0f kB -> %-24s %7.0f kB\n",
		filepath.Base(path), kb(len(raw)), filepath.Base(target), kb(out.Len()))

	if dry {
		return int64(len(raw)), int64(out.Len()), nil
	}

	err = os.WriteFile(target, out.Bytes(), 0o644) //nolint:gosec // checked-in web asset
	if err != nil {
		return 0, 0, err
	}

	if target != path {
		err := os.Remove(path)
		if err != nil {
			return 0, 0, err
		}
	}

	return int64(len(raw)), int64(out.Len()), nil
}

func downscale(img image.Image, maxW int) image.Image {
	b := img.Bounds()
	if maxW <= 0 || b.Dx() <= maxW {
		return img
	}

	h := b.Dy() * maxW / b.Dx()
	dst := image.NewRGBA(image.Rect(0, 0, maxW, h))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), img, b, draw.Src, nil)

	return dst
}

// opaque reports whether every pixel is fully opaque.
func opaque(img image.Image) bool {
	if o, ok := img.(interface{ Opaque() bool }); ok {
		return o.Opaque()
	}

	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if _, _, _, a := img.At(x, y).RGBA(); a != 0xffff {
				return false
			}
		}
	}

	return true
}

func kb(n int) float64   { return float64(n) / 1024 }
func mb(n int64) float64 { return float64(n) / (1024 * 1024) }
