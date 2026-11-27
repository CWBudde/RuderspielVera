// fit [-bits N] [-q Q] <in.png> <out.png> <w> <h> [cx cy]
//
// Scales a PNG so it covers w×h (aspect kept), then crops the overflow to exactly w×h and writes a
// PNG, or a JPEG when <out> ends in .jpg/.jpeg (-q quality, default 85). cx, cy (0..1, default
// 0.5 0.5) place the crop window: 0 keeps the left/top edge, 1 the right/bottom edge. Uses CatmullRom resampling. Prints the source size, scale, crop box and file size.
//
// -bits N (1..8, default 8) rounds each colour channel to N significant bits before encoding, which
// makes painted art compress much better as PNG; 6 is not visibly different, 5 may band in skies.
// Dense, detailed art (the board) stays over 1 MB as PNG: write it as .jpg.
package main

import (
	"flag"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/image/draw"
)

func die(f string, a ...any) { fmt.Fprintf(os.Stderr, "fit: "+f+"\n", a...); os.Exit(1) }

func main() {
	bits := flag.Int("bits", 8, "significant bits per colour channel (1..8)")
	quality := flag.Int("q", 85, "JPEG quality (1..100) for .jpg output")
	flag.Parse()
	args := flag.Args()
	if len(args) != 4 && len(args) != 6 {
		die("usage: fit [-bits N] [-q Q] <in.png> <out.png> <w> <h> [cx cy]")
	}
	if *bits < 1 || *bits > 8 {
		die("-bits must be 1..8")
	}
	in, outPath := args[0], args[1]
	w, err1 := strconv.Atoi(args[2])
	h, err2 := strconv.Atoi(args[3])
	if err1 != nil || err2 != nil || w <= 0 || h <= 0 {
		die("bad size %s x %s", args[2], args[3])
	}
	cx, cy := 0.5, 0.5
	if len(args) == 6 {
		var e1, e2 error
		cx, e1 = strconv.ParseFloat(args[4], 64)
		cy, e2 = strconv.ParseFloat(args[5], 64)
		if e1 != nil || e2 != nil || cx < 0 || cx > 1 || cy < 0 || cy > 1 {
			die("cx cy must be in 0..1")
		}
	}
	f, err := os.Open(in)
	if err != nil {
		die("%v", err)
	}
	src, err := png.Decode(f)
	f.Close()
	if err != nil {
		die("decode %s: %v", in, err)
	}
	sb := src.Bounds()
	sw, sh := sb.Dx(), sb.Dy()
	// Cover: the larger of the two scale factors.
	s := float64(w) / float64(sw)
	if t := float64(h) / float64(sh); t > s {
		s = t
	}
	// Source window that maps onto the target.
	ww := min(int(float64(w)/s+0.5), sw)
	wh := min(int(float64(h)/s+0.5), sh)
	x0 := sb.Min.X + int(float64(sw-ww)*cx+0.5)
	y0 := sb.Min.Y + int(float64(sh-wh)*cy+0.5)
	win := image.Rect(x0, y0, x0+ww, y0+wh)
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, win, draw.Src, nil)
	if *bits < 8 {
		quantize(dst.Pix, uint(8-*bits))
	}
	out, err := os.Create(outPath)
	if err != nil {
		die("%v", err)
	}
	switch strings.ToLower(filepath.Ext(outPath)) {
	case ".jpg", ".jpeg":
		err = jpeg.Encode(out, dst, &jpeg.Options{Quality: *quality})
	default:
		enc := png.Encoder{CompressionLevel: png.BestCompression}
		err = enc.Encode(out, dst)
	}
	if err != nil {
		die("encode: %v", err)
	}
	if err := out.Close(); err != nil {
		die("%v", err)
	}
	st, err := os.Stat(outPath)
	if err != nil {
		die("%v", err)
	}
	fmt.Printf("%dx%d -> scale %.4f, crop %d,%d %dx%d -> %dx%d %s (%d KB)\n",
		sw, sh, s, x0, y0, ww, wh, w, h, outPath, st.Size()/1024)
}

// quantize rounds the colour channels (not alpha) of an RGBA buffer, dropping the low `drop` bits.
func quantize(pix []uint8, drop uint) {
	step := 1 << drop
	top := 255 &^ (step - 1)
	for i := range pix {
		if i%4 == 3 {
			continue
		}
		v := (int(pix[i]) + step/2) >> drop << drop
		pix[i] = uint8(min(v, top))
	}
}
