package tiles

import (
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gofiber/fiber/v2"
)

// FontHandler serves font glyphs for map text rendering
// Serves from local storage if available, otherwise downloads and caches
func FontHandler(c *fiber.Ctx) error {
	fontstack := c.Params("fontstack")
	rangeParam := c.Params("range")

	// Define local font directory
	fontDir := "./public/fonts"
	fontPath := filepath.Join(fontDir, fontstack, rangeParam+".pbf")

	// Check if font exists locally
	if _, err := os.Stat(fontPath); err == nil {
		// Serve local font file
		c.Set("Content-Type", "application/x-protobuf")
		c.Set("Cache-Control", "public, max-age=86400") // Cache for 1 day
		return c.SendFile(fontPath)
	}

	// Font not found locally, download from OpenMapTiles
	fontURL := "https://fonts.openmaptiles.org/" + fontstack + "/" + rangeParam + ".pbf"

	resp, err := http.Get(fontURL)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Error fetching font")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return c.Status(resp.StatusCode).SendString("Font not found")
	}

	// Read font data
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Error reading font data")
	}

	// Save to local storage for future requests
	os.MkdirAll(filepath.Dir(fontPath), os.ModePerm)
	os.WriteFile(fontPath, body, 0644)

	// Serve the font
	c.Set("Content-Type", "application/x-protobuf")
	c.Set("Cache-Control", "public, max-age=86400")
	return c.Send(body)
}
