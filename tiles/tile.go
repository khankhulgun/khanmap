package tiles

import (
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/khankhulgun/khanmap/maplayer"
	"github.com/khankhulgun/khanmap/models"
	"github.com/lambda-platform/lambda/DB"
	"log"
	"math"
	"os"
	"path/filepath"
	"strconv"
)

type Feature struct {
	Geom []byte `gorm:"column:geom"`
	ID   int    `gorm:"column:id"`
	Name string `gorm:"column:name"`
}

const (
	tileSize   = 4096
	tileExtent = 256
)

func tileToBBox(z, x, y int) (minX, minY, maxX, maxY float64) {
	n := 1 << z
	nf := float64(n)
	minX = float64(x)/nf*360.0 - 180.0
	minY = math.Atan(math.Sinh(math.Pi*(1-2*float64(y)/nf))) * 180.0 / math.Pi
	maxX = float64(x+1)/nf*360.0 - 180.0
	maxY = math.Atan(math.Sinh(math.Pi*(1-2*float64(y+1)/nf))) * 180.0 / math.Pi
	return
}

func parseTileParams(c *fiber.Ctx) (int, int, int, error) {
	z, err := strconv.Atoi(c.Params("z"))
	if err != nil {
		return 0, 0, 0, err
	}
	x, err := strconv.Atoi(c.Params("x"))
	if err != nil {
		return 0, 0, 0, err
	}
	y, err := strconv.Atoi(c.Params("y"))
	if err != nil {
		return 0, 0, 0, err
	}
	return z, x, y, nil
}

func fetchTileData(query string, args ...interface{}) ([]byte, error) {
	var mvtData []byte
	err := DB.DB.Raw(query, args...).Row().Scan(&mvtData)
	if err != nil {
		return nil, err
	}
	return mvtData, nil
}

func tileHandler(layer models.MapLayersForTile) fiber.Handler {
	return func(c *fiber.Ctx) error {
		z, x, y, err := parseTileParams(c)
		if err != nil {
			log.Printf("Invalid tile parameters: %v", err)
			return c.Status(fiber.StatusBadRequest).SendString("Invalid tile parameters")
		}
		mvtData, err := getVectorTile(z, x, y, layer)

		if err != nil {
			log.Printf("Database error: %v", err)
			return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
		}

		c.Set("Content-Type", "application/vnd.mapbox-vector-tile")
		return c.Send(mvtData)
	}
}
func getVectorTile(z, x, y int, layer models.MapLayersForTile) ([]byte, error) {
	minX, minY, maxX, maxY := tileToBBox(z, x, y)
	sqlColumns := maplayer.ConstructSQLColumns(layer, true)

	query := `
			SELECT ST_AsMVT(q, ?, ?, ?) FROM (
				SELECT ` + sqlColumns + `, ST_AsMVTGeom(
					` + layer.GeometryFieldName + `,
					ST_MakeEnvelope(?, ?, ?, ?, 4326),
					?,
					?,
					true
				) AS ` + layer.GeometryFieldName + `
				FROM ` + layer.DbSchema + `.` + layer.DbTable + `
				WHERE ` + layer.GeometryFieldName + ` && ST_MakeEnvelope(?, ?, ?, ?, 4326)
			) AS q
		`
	return fetchTileData(query, layer.DbSchema+"."+layer.DbTable, tileSize, layer.GeometryFieldName, minX, minY, maxX, maxY, tileSize, tileExtent, minX, minY, maxX, maxY)

}
func SaveVectorTileHandler(c *fiber.Ctx) error {
	layer := c.Params("layer")
	z := c.Params("z")
	x := c.Params("x")
	y := c.Params("y")

	// Fetch the layer details
	layerDetails, err := maplayer.FetchLayerDetails(layer)
	if err != nil {
		log.Printf("Layer not found: %v", err)
		return c.Status(fiber.StatusNotFound).SendString("Layer not found")
	}

	// Construct the tile file path based on the layer, zoom, x, y parameters
	tilePath := filepath.Join(downloadDir, fmt.Sprintf("%s/%s/%s/%s.pbf", layerDetails.ID, z, x, y))

	// Check if the tile already exists
	if _, err := os.Stat(tilePath); err == nil {
		// Tile exists, serve the existing tile
		return c.SendFile(tilePath)
	}

	return tileHandler(layerDetails)(c)
}

func SaveHandler(c *fiber.Ctx) error {
	layer := c.Params("layer")

	layerDetails, err := maplayer.FetchLayerDetails(layer)
	if err != nil {
		log.Printf("Layer not found: %v", err)
		return c.Status(fiber.StatusNotFound).SendString("Layer not found")
	}
	CreateTiles(layerDetails)

	return nil
}

func VectorTileHandler(c *fiber.Ctx) error {
	layer := c.Params("layer")
	layerDetails, err := maplayer.FetchLayerDetails(layer)
	if err != nil {
		log.Printf("Layer not found: %v", err)
		return c.Status(fiber.StatusNotFound).SendString("Layer not found")
	}
	return tileHandler(layerDetails)(c)
}
