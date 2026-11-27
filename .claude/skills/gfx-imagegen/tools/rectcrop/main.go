package main

// rectcrop <in.png> <out.png> <x> <y> <w> <h>: copies a rectangle of any PNG.
import (
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"os"
	"strconv"
)

func main() {
	f, err := os.Open(os.Args[1])
	if err != nil {
		panic(err)
	}
	src, err := png.Decode(f)
	if err != nil {
		panic(err)
	}
	n := func(i int) int { v, _ := strconv.Atoi(os.Args[i]); return v }
	r := image.Rect(n(3), n(4), n(3)+n(5), n(4)+n(6)).Add(src.Bounds().Min).Intersect(src.Bounds())
	out := image.NewNRGBA(image.Rect(0, 0, r.Dx(), r.Dy()))
	draw.Draw(out, out.Bounds(), src, r.Min, draw.Src)
	o, err := os.Create(os.Args[2])
	if err != nil {
		panic(err)
	}
	defer o.Close()
	if err := png.Encode(o, out); err != nil {
		panic(err)
	}
	fmt.Println(os.Args[2], src.Bounds(), "->", r)
}
