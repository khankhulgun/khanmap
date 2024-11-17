package sprite

import (
	"encoding/json"
	"github.com/khankhulgun/khanmap/models"
	"image"
	"image/draw"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func MakeSprite(srcDir, destFile string) {

	files, err := filepath.Glob(filepath.Join(srcDir, "*.png"))
	if err != nil {
		log.Fatalf("Failed to read files: %v", err)
	}

	if len(files) == 0 {
		log.Fatalf("No PNG files found in %s", srcDir)
	}

	// Load images and calculate sprite dimensions
	var images []image.Image
	var spriteWidth, maxHeight int
	spriteMeta := make(map[string]models.SpriteMeta)

	for _, file := range files {
		f, err := os.Open(file)
		if err != nil {
			log.Fatalf("Failed to open file: %v", err)
		}
		img, err := png.Decode(f)
		f.Close()
		if err != nil {
			log.Fatalf("Failed to decode PNG: %v", err)
		}

		images = append(images, img)
		bounds := img.Bounds()
		width, height := bounds.Dx(), bounds.Dy()
		name := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file)) // Remove file extension
		spriteMeta[name] = models.SpriteMeta{
			X:          spriteWidth,
			Y:          0,
			Width:      width,
			Height:     height,
			PixelRatio: 1,
		}
		spriteWidth += width
		if height > maxHeight {
			maxHeight = height
		}
	}

	// Create the sprite sheet image
	spriteImg := image.NewRGBA(image.Rect(0, 0, spriteWidth, maxHeight))
	currentX := 0
	for _, img := range images {
		bounds := img.Bounds()
		width, height := bounds.Dx(), bounds.Dy()
		draw.Draw(spriteImg, image.Rect(currentX, 0, currentX+width, height), img, image.Point{}, draw.Over)
		currentX += width
	}

	// Save the sprite sheet as PNG
	saveImage(spriteImg, destFile+".png")
	saveImage(spriteImg, destFile+"@2x.png")

	// Save the JSON metadata
	saveJSON(spriteMeta, destFile+".json")
	saveJSON(spriteMeta, destFile+"@2x.json")

	log.Printf("Sprite sheet and JSON metadata created: %s, %s", destFile+".png", destFile+".json")
}

func saveImage(img image.Image, filename string) {
	outFile, err := os.Create(filename)
	if err != nil {
		log.Fatalf("Failed to create sprite image: %v", err)
	}
	defer outFile.Close()
	if err := png.Encode(outFile, img); err != nil {
		log.Fatalf("Failed to encode PNG: %v", err)
	}

}

func saveJSON(meta map[string]models.SpriteMeta, filename string) {
	jsonFile, err := os.Create(filename)
	if err != nil {
		log.Fatalf("Failed to create JSON file: %v", err)
	}
	defer jsonFile.Close()
	encoder := json.NewEncoder(jsonFile)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(meta); err != nil {
		log.Fatalf("Failed to encode JSON: %v", err)
	}
}
