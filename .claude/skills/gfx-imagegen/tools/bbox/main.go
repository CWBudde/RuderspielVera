// bbox prints the alpha>128 bounding box per grid cell: bbox in.png cols rows
package main

import (
	"fmt"
	"image/color"
	"image/png"
	"os"
	"strconv"
)

func main() {
	f, _ := os.Open(os.Args[1])
	img, err := png.Decode(f)
	if err != nil {
		panic(err)
	}
	gx, _ := strconv.Atoi(os.Args[2])
	gy, _ := strconv.Atoi(os.Args[3])
	b := img.Bounds()
	cw, ch := b.Dx()/gx, b.Dy()/gy
	for j := 0; j < gy; j++ {
		for i := 0; i < gx; i++ {
			x0, y0, x1, y1, n, edge := 1<<30, 1<<30, -1, -1, 0, 0
			for y := j * ch; y < (j+1)*ch; y++ {
				for x := i * cw; x < (i+1)*cw; x++ {
					if color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA).A > 128 {
						n++
						x0, y0, x1, y1 = min(x0, x), min(y0, y), max(x1, x), max(y1, y)
						if x == i*cw || x == (i+1)*cw-1 || y == j*ch || y == (j+1)*ch-1 {
							edge++
						}
					}
				}
			}
			if n == 0 {
				fmt.Printf("cell %d,%d empty\n", j, i)
				continue
			}
			fmt.Printf("cell r%d c%d: w=%d h=%d (cell-rel %d,%d) cover=%.1f%% touches-edge=%d\n", j, i, x1-x0+1, y1-y0+1, x0-i*cw, y0-j*ch, 100*float64(n)/float64(cw*ch), edge)
		}
	}
}
