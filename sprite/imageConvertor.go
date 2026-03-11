package sprite

import (
	"fmt"
	"image"
	"image/png"
	"log"
	"os"

	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
)

func SVGToPNG(svgFile, pngFile string) error {
	// Open the SVG file
	f, err := os.Open(svgFile)
	if err != nil {
		log.Printf("SVGToPNG: failed to open SVG file %s: %v", svgFile, err)
		return fmt.Errorf("failed to open SVG file %s: %w", svgFile, err)
	}
	defer f.Close()

	// Parse the SVG
	icon, err := oksvg.ReadIconStream(f)
	if err != nil {
		log.Printf("SVGToPNG: failed to parse SVG file %s: %v", svgFile, err)
		return fmt.Errorf("failed to parse SVG file %s: %w", svgFile, err)
	}

	// Set the target size: fixed width of 30px, height scales proportionally
	targetWidth := 72
	aspectRatio := icon.ViewBox.H / icon.ViewBox.W
	w := targetWidth
	h := int(float64(targetWidth) * aspectRatio)
	icon.SetTarget(0, 0, float64(w), float64(h))

	// Create a new RGBA image with full transparency
	img := image.NewRGBA(image.Rect(0, 0, w, h))

	// Configure the Dasher and Scanner
	scanner := rasterx.NewScannerGV(w, h, img, img.Bounds())
	dasher := rasterx.NewDasher(w, h, scanner)
	dasher.SetColor(nil) // Ensure transparency is respected

	// Draw the SVG onto the image
	icon.Draw(dasher, 1)

	// Encode the image as PNG with full transparency
	out, err := os.Create(pngFile)
	if err != nil {
		log.Printf("SVGToPNG: failed to create output PNG %s: %v", pngFile, err)
		return fmt.Errorf("failed to create output PNG %s: %w", pngFile, err)
	}
	defer out.Close()

	err = png.Encode(out, img)
	if err != nil {
		log.Printf("SVGToPNG: failed to encode PNG %s: %v", pngFile, err)
		return fmt.Errorf("failed to encode PNG %s: %w", pngFile, err)
	}

	log.Printf("SVGToPNG: successfully converted %s -> %s", svgFile, pngFile)
	return nil
}
