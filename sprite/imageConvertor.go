package sprite

import (
	"image"
	"image/png"
	"os"

	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
)

func SVGToPNG(svgFile, pngFile string) error {
	// Open the SVG file
	f, err := os.Open(svgFile)
	if err != nil {
		return err
	}
	defer f.Close()

	// Parse the SVG
	icon, err := oksvg.ReadIconStream(f)
	if err != nil {
		return err
	}

	// Set the target size (adjust as needed)
	w := int(icon.ViewBox.W) * 2
	h := int(icon.ViewBox.H) * 2
	icon.SetTarget(0, 0, float64(w), float64(h))

	// Create a new RGBA image with full transparency
	img := image.NewRGBA(image.Rect(0, 0, w, h))

	// Configure the Dasher and Scanner
	scanner := rasterx.NewScannerGV(w, h, img, img.Bounds())
	dasher := rasterx.NewDasher(w, h, scanner)
	dasher.SetColor(nil) // Ensure transparency is respected

	// Draw the SVG onto the image
	icon.Draw(dasher, 1)

	// Remove white background (make white pixels transparent)
	bounds := img.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			// Check for white pixel (allowing for slight variations/alpha premultiplication)
			// RGBA() returns color in [0, 65535] range
			if r > 65000 && g > 65000 && b > 65000 && a > 65000 {
				img.Set(x, y, image.Transparent)
			}
		}
	}

	// Encode the image as PNG with full transparency
	out, err := os.Create(pngFile)
	if err != nil {
		return err
	}
	defer out.Close()

	err = png.Encode(out, img)
	if err != nil {
		return err
	}

	return nil
}
