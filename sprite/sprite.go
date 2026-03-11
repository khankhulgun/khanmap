package sprite

import (
	"encoding/json"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/khankhulgun/khanmap/models"
)

func MakeSprite(srcDir, destFile string) error {

	files, err := filepath.Glob(filepath.Join(srcDir, "*.png"))
	if err != nil {
		log.Printf("MakeSprite: failed to read files in %s: %v", srcDir, err)
		return fmt.Errorf("failed to read files in %s: %w", srcDir, err)
	}

	if len(files) == 0 {
		log.Printf("MakeSprite: no PNG files found in %s", srcDir)
		return fmt.Errorf("no PNG files found in %s", srcDir)
	}

	// Load images and calculate sprite dimensions
	var images []image.Image
	var spriteWidth, maxHeight int
	spriteMeta := make(map[string]models.SpriteMeta)

	for _, file := range files {
		f, err := os.Open(file)
		if err != nil {
			log.Printf("MakeSprite: failed to open file %s: %v", file, err)
			return fmt.Errorf("failed to open file %s: %w", file, err)
		}
		img, err := png.Decode(f)
		f.Close()
		if err != nil {
			log.Printf("MakeSprite: failed to decode PNG %s: %v", file, err)
			return fmt.Errorf("failed to decode PNG %s: %w", file, err)
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
			PixelRatio: 2,
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
	if err := saveImage(spriteImg, destFile+".png"); err != nil {
		return fmt.Errorf("failed to save sprite image %s.png: %w", destFile, err)
	}
	if err := saveImage(spriteImg, destFile+"@2x.png"); err != nil {
		return fmt.Errorf("failed to save sprite image %s@2x.png: %w", destFile, err)
	}

	// Save the JSON metadata
	if err := saveJSON(spriteMeta, destFile+".json"); err != nil {
		return fmt.Errorf("failed to save sprite JSON %s.json: %w", destFile, err)
	}
	if err := saveJSON(spriteMeta, destFile+"@2x.json"); err != nil {
		return fmt.Errorf("failed to save sprite JSON %s@2x.json: %w", destFile, err)
	}

	log.Printf("Sprite sheet and JSON metadata created: %s, %s", destFile+".png", destFile+".json")
	return nil
}

func saveImage(img image.Image, filename string) error {
	outFile, err := os.Create(filename)
	if err != nil {
		log.Printf("saveImage: failed to create file %s: %v", filename, err)
		return fmt.Errorf("failed to create sprite image %s: %w", filename, err)
	}
	defer outFile.Close()
	if err := png.Encode(outFile, img); err != nil {
		log.Printf("saveImage: failed to encode PNG %s: %v", filename, err)
		return fmt.Errorf("failed to encode PNG %s: %w", filename, err)
	}
	return nil
}

func saveJSON(meta map[string]models.SpriteMeta, filename string) error {
	jsonFile, err := os.Create(filename)
	if err != nil {
		log.Printf("saveJSON: failed to create file %s: %v", filename, err)
		return fmt.Errorf("failed to create JSON file %s: %w", filename, err)
	}
	defer jsonFile.Close()
	encoder := json.NewEncoder(jsonFile)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(meta); err != nil {
		log.Printf("saveJSON: failed to encode JSON %s: %v", filename, err)
		return fmt.Errorf("failed to encode JSON %s: %w", filename, err)
	}
	return nil
}
