// comp composites an RGBA PNG over a flat colour: comp in.png out.png r g b
package main

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"strconv"
)

func main() {
	f, _ := os.Open(os.Args[1])
	src, err := png.Decode(f)
	if err != nil {
		panic(err)
	}
	r, _ := strconv.Atoi(os.Args[3])
	g, _ := strconv.Atoi(os.Args[4])
	b, _ := strconv.Atoi(os.Args[5])
	bd := src.Bounds()
	dst := image.NewRGBA(bd)
	for y := bd.Min.Y; y < bd.Max.Y; y++ {
		for x := bd.Min.X; x < bd.Max.X; x++ {
			c := color.NRGBAModel.Convert(src.At(x, y)).(color.NRGBA)
			a := int(c.A)
			mix := func(s uint8, bg int) uint8 { return uint8((int(s)*a + bg*(255-a)) / 255) }
			dst.Set(x, y, color.RGBA{mix(c.R, r), mix(c.G, g), mix(c.B, b), 255})
		}
	}
	o, _ := os.Create(os.Args[2])
	png.Encode(o, dst)
	o.Close()
}
