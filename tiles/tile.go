package tiles

import (
	"github.com/dgraph-io/ristretto"
	"github.com/gofiber/fiber/v2"
	"github.com/khankhulgun/khanmap/models"
	"github.com/lambda-platform/lambda/DB"
	"log"
	"math"
	"strconv"
	"strings"
	"time"
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

var layerCache *ristretto.Cache

func init() {
	// Initialize the cache with Ristretto
	var err error
	layerCache, err = ristretto.NewCache(&ristretto.Config{
		NumCounters: 1e7,     // number of keys to track frequency of (10M)
		MaxCost:     1 << 30, // maximum cost of cache (1GB)
		BufferItems: 64,      // number of keys per Get buffer
	})
	if err != nil {
		log.Fatalf("Failed to initialize cache: %v", err)
	}
}

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

func fetchLayerDetails(layerID string) (models.MapLayersForTile, error) {
	layerID = strings.TrimSpace(layerID)

	if cachedLayer, found := layerCache.Get(layerID); found {
		layerDetails, ok := cachedLayer.(models.MapLayersForTile)
		if ok {
			return layerDetails, nil
		}
	}

	var layerDetails models.MapLayersForTile
	err := DB.DB.Where("id = ?", layerID).First(&layerDetails).Error
	if err != nil {
		return layerDetails, err
	}

	layerCache.SetWithTTL(layerID, layerDetails, 1, 60*time.Minute)
	layerCache.Wait()

	return layerDetails, nil
}
func constructSQLColumns(layer models.MapLayersForTile) string {
	sqlColumns := layer.ColumnSelects
	if sqlColumns == "" {
		return "'" + layer.IDFieldName + "'"
	}

	columns := strings.Split(sqlColumns, ",")
	columnMap := make(map[string]bool)
	idPresent := false
	uniqueValueFieldFound := false

	for _, col := range columns {
		col = strings.TrimSpace(col)
		if col != "" {
			columnMap[col] = true
			if col == layer.IDFieldName {
				idPresent = true
			}
			if layer.UniqueValueField != nil && col == *layer.UniqueValueField {
				uniqueValueFieldFound = true
			}
		}
	}

	if !idPresent {
		columnMap[layer.IDFieldName] = true
	}

	delete(columnMap, layer.GeometryFieldName)

	var newColumns []string
	for col := range columnMap {
		newColumns = append(newColumns, "\""+col+"\"")
	}

	if layer.UniqueValueField != nil && !uniqueValueFieldFound {
		newColumns = append(newColumns, "\""+*layer.UniqueValueField+"\"")
	}

	return strings.Join(newColumns, ", ")
}

func tileHandler(layer models.MapLayersForTile) fiber.Handler {
	return func(c *fiber.Ctx) error {
		z, x, y, err := parseTileParams(c)
		if err != nil {
			log.Printf("Invalid tile parameters: %v", err)
			return c.Status(fiber.StatusBadRequest).SendString("Invalid tile parameters")
		}

		minX, minY, maxX, maxY := tileToBBox(z, x, y)
		sqlColumns := constructSQLColumns(layer)

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
		mvtData, err := fetchTileData(query, layer.DbSchema+"."+layer.DbTable, tileSize, layer.GeometryFieldName, minX, minY, maxX, maxY, tileSize, tileExtent, minX, minY, maxX, maxY)
		if err != nil {
			log.Printf("Database error: %v", err)
			return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
		}

		c.Set("Content-Type", "application/vnd.mapbox-vector-tile")
		return c.Send(mvtData)
	}
}

func VectorTileHandler(c *fiber.Ctx) error {
	layer := c.Params("layer")
	layerDetails, err := fetchLayerDetails(layer)
	if err != nil {
		log.Printf("Layer not found: %v", err)
		return c.Status(fiber.StatusNotFound).SendString("Layer not found")
	}
	return tileHandler(layerDetails)(c)
}
