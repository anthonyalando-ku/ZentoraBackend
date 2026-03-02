package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"

	"github.com/chai2010/webp"
)

func main() {
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: color.White}, image.Point{}, draw.Src)

	var buf bytes.Buffer
	err := webp.Encode(&buf, img, &webp.Options{Quality: 75})
	if err != nil {
		fmt.Println("Encode failed:", err)
		return
	}
	fmt.Println("WebP encode succeeded, bytes:", buf.Len())
}