package tiles

import (
	"errors"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/khankhulgun/khanmap/maplayer"
	"github.com/khankhulgun/khanmap/models"
	"github.com/lambda-platform/lambda/DB"
	agentUtils "github.com/lambda-platform/lambda/agent/utils"
	"log"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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

func tileHandler(layer models.MapLayersForTile, user interface{}, filters map[string]string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		z, x, y, err := parseTileParams(c)
		if err != nil {
			log.Printf("Invalid tile parameters: %v", err)
			return c.Status(fiber.StatusBadRequest).SendString("Invalid tile parameters")
		}

		mvtData, err := getVectorTile(z, x, y, layer, user, filters)
		if err != nil {
			log.Printf("Database error: %v", err)
			return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
		}

		c.Set("Content-Type", "application/vnd.mapbox-vector-tile")
		return c.Send(mvtData)
	}
}

func getVectorTile(z, x, y int, layer models.MapLayersForTile, user interface{}, adminFilters map[string]string) ([]byte, error) {
	minX, minY, maxX, maxY := tileToBBox(z, x, y)
	sqlColumns := maplayer.ConstructSQLColumns(layer, true)

	rawSQL := `
		SELECT ST_AsMVT(q, ?, ?, ?) FROM (
			SELECT ` + sqlColumns + `, ST_AsMVTGeom(
				` + layer.GeometryFieldName + `,
				ST_MakeEnvelope(?, ?, ?, ?, 4326),
				?,
				?,
				true
			) AS ` + layer.GeometryFieldName + `
			FROM ` + layer.DbSchema + `.` + layer.DbTable + `
			WHERE ` + layer.GeometryFieldName + ` && ST_MakeEnvelope(?, ?, ?, ?, 4326) %s
		) AS q
	`

	userMap, _ := user.(map[string]interface{})

	query := rawSQL
	var filterConditions []string
	var filterValues []interface{}

	if layer.IsPermission {
		if len(layer.Permissions) > 0 {
			roleVal, ok := userMap["role"]
			roleFloat, isFloat := roleVal.(float64)
			roleInt := int(roleFloat)
			if !ok || !isFloat {
				return nil, errors.New("user role is missing or not a float")
			}

			hasPermission := false
			for _, perm := range layer.Permissions {
				if roleInt == perm.RoleID {
					hasPermission = true
					break
				}
			}
			if !hasPermission {
				return nil, errors.New("user does not have permission for this layer")
			}
		}

		for _, filter := range layer.Filters {
			val, ok := userMap[filter.UserColumn]
			if !ok {
				continue
			}
			filterConditions = append(filterConditions, fmt.Sprintf("AND %s = ?", filter.TableColumn))
			filterValues = append(filterValues, val)
		}
	}

	for key, value := range adminFilters {
		filterConditions = append(filterConditions, fmt.Sprintf("AND %s = ?", key))
		filterValues = append(filterValues, value)
	}

	query = fmt.Sprintf(query, strings.Join(filterConditions, " "))

	args := []interface{}{
		layer.DbSchema + "." + layer.DbTable,
		tileSize,
		layer.GeometryFieldName,
		minX, minY, maxX, maxY,
		tileSize,
		tileExtent,
		minX, minY, maxX, maxY,
	}
	args = append(args, filterValues...)

	return fetchTileData(query, args...)
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

	return tileHandler(layerDetails, nil, nil)(c)
}

func SaveHandler(c *fiber.Ctx) error {
	layer := c.Params("layer")

	layerDetails, err := maplayer.FetchLayerDetails(layer)
	if err != nil {
		log.Printf("Layer not found: %v", err)
		return c.Status(fiber.StatusNotFound).SendString("Layer not found")
	}
	createErr := CreateTiles(layerDetails)
	if createErr != nil {
		log.Printf("Error creating tiles: %v", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Error creating tiles")
	}

	return nil
}

func VectorTileHandler(c *fiber.Ctx) error {
	layer := c.Params("layer")
	query := c.Queries()

	filters := make(map[string]string)
	for key, value := range query {
		filters[key] = value
	}

	layerDetails, err := maplayer.FetchLayerDetails(layer)
	if err != nil {
		log.Printf("Layer not found: %v", err)
		return c.Status(fiber.StatusNotFound).SendString("Layer not found")
	}

	return tileHandler(layerDetails, nil, filters)(c)
}

func VectorTileHandlerWithPermission(c *fiber.Ctx) error {
	layer := c.Params("layer")
	user, err := agentUtils.AuthUserObject(c)
	if err != nil {
		log.Printf("User not found: %v", err)
		return c.Status(fiber.StatusUnauthorized).SendString("User not found")
	}

	query := c.Queries()
	filters := make(map[string]string)
	for key, value := range query {
		filters[key] = value
	}

	layerDetails, err := maplayer.FetchLayerDetails(layer)
	if err != nil {
		log.Printf("Layer not found: %v", err)
		return c.Status(fiber.StatusNotFound).SendString("Layer not found")
	}

	return tileHandler(layerDetails, user, filters)(c)
}
