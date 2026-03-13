package sprite

import (
	"fmt"
	"image"
	"image/png"
	"log"
	"os"
	"os/exec"

	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
)

// SVGToPNG converts an SVG file to a PNG file.
// It first tries external tools (rsvg-convert, resvg) which have full
// gradient support (linearGradient, radialGradient). If none are available,
// it falls back to the Go-native oksvg/rasterx renderer.
func SVGToPNG(svgFile, pngFile string) error {
	// Strategy 1: Try rsvg-convert (librsvg) — full gradient support
	if path, err := exec.LookPath("rsvg-convert"); err == nil {
		cmd := exec.Command(path, "-w", "72", "-f", "png", "-o", pngFile, svgFile)
		if err := cmd.Run(); err == nil {
			log.Printf("SVGToPNG [rsvg-convert]: successfully converted %s -> %s", svgFile, pngFile)
			return nil
		}
		log.Printf("SVGToPNG: rsvg-convert failed for %s, trying next method: %v", svgFile, err)
	}

	// Strategy 2: Try resvg — full gradient support
	if path, err := exec.LookPath("resvg"); err == nil {
		cmd := exec.Command(path, "--width", "72", svgFile, pngFile)
		if err := cmd.Run(); err == nil {
			log.Printf("SVGToPNG [resvg]: successfully converted %s -> %s", svgFile, pngFile)
			return nil
		}
		log.Printf("SVGToPNG: resvg failed for %s, trying next method: %v", svgFile, err)
	}

	// Strategy 3: Fall back to Go-native oksvg/rasterx
	return svgToPNGNative(svgFile, pngFile)
}

// svgToPNGNative uses the oksvg + rasterx libraries to render SVG to PNG.
// Note: oksvg has partial gradient support. Complex gradients may render
// as black. For full gradient support, install rsvg-convert or resvg.
func svgToPNGNative(svgFile, pngFile string) error {
	// Open the SVG file
	f, err := os.Open(svgFile)
	if err != nil {
		log.Printf("SVGToPNG: failed to open SVG file %s: %v", svgFile, err)
		return fmt.Errorf("failed to open SVG file %s: %w", svgFile, err)
	}
	defer f.Close()

	// Parse the SVG — use WarnErrorMode to not abort on unsupported elements
	icon, err := oksvg.ReadIconStream(f, oksvg.WarnErrorMode)
	if err != nil {
		log.Printf("SVGToPNG: failed to parse SVG file %s: %v", svgFile, err)
		return fmt.Errorf("failed to parse SVG file %s: %w", svgFile, err)
	}

	// Set the target size: fixed width of 72px, height scales proportionally
	targetWidth := 72
	aspectRatio := icon.ViewBox.H / icon.ViewBox.W
	w := targetWidth
	h := int(float64(targetWidth) * aspectRatio)
	if h < 1 {
		h = 1
	}
	icon.SetTarget(0, 0, float64(w), float64(h))

	// Create a new RGBA image with full transparency
	img := image.NewRGBA(image.Rect(0, 0, w, h))

	// Configure the Scanner, Filler and Dasher
	// Using both Filler (for fills/gradients) and Dasher (for strokes)
	scanner := rasterx.NewScannerGV(w, h, img, img.Bounds())
	dasher := rasterx.NewDasher(w, h, scanner)

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

	log.Printf("SVGToPNG [native]: successfully converted %s -> %s", svgFile, pngFile)
	return nil
}
