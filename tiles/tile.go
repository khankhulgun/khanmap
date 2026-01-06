package tiles

import (
	"errors"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/khankhulgun/khanmap/maplayer"
	"github.com/khankhulgun/khanmap/models"
	"github.com/lambda-platform/lambda/DB"
	agentUtils "github.com/lambda-platform/lambda/agent/utils"
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

// getClusterRadius calculates the cluster radius in degrees based on zoom level and latitude
// standardRadiusPixels: default radius in pixels (e.g., 40-60)
func getClusterRadius(zoom int, lat float64) float64 {
	// Earth circumference ~40,075,017 meters
	// Resolution (meters/pixel) = 156543.03 * cos(lat) / 2^zoom

	const standardRadiusPixels = 50.0
	// Convert latitude to radians
	latRad := lat * math.Pi / 180.0
	resolution := 156543.03 * math.Cos(latRad) / math.Pow(2, float64(zoom))

	// Radius in meters
	radiusMeters := standardRadiusPixels * resolution

	// Convert to degrees (1 degree ~ 111,320m)
	return radiusMeters / 111320.0
}

func tileHandler(layer models.MapLayersForTile, user interface{}, filters map[string]string, areaFilters map[string]string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		z, x, y, err := parseTileParams(c)
		if err != nil {
			log.Printf("Invalid tile parameters: %v", err)
			return c.Status(fiber.StatusBadRequest).SendString("Invalid tile parameters")
		}

		mvtData, err := getVectorTile(z, x, y, layer, user, filters, areaFilters)
		if err != nil {
			log.Printf("Database error: %v", err)
			return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
		}

		c.Set("Content-Type", "application/vnd.mapbox-vector-tile")
		return c.Send(mvtData)
	}
}

func getVectorTile(z, x, y int, layer models.MapLayersForTile, user interface{}, adminFilters map[string]string, areaFilters map[string]string) ([]byte, error) {
	minX, minY, maxX, maxY := tileToBBox(z, x, y)

	// Calculate buffer in degrees to prevent edge artifacts
	// Default buffer is often 256 units in a 4096 unit tile (approx 6.25%)
	const bufferRatio = 256.0 / 4096.0
	xSpan := maxX - minX
	ySpan := maxY - minY
	bufferX := xSpan * bufferRatio
	bufferY := ySpan * bufferRatio

	// Buffered BBox for SQL selection
	bMinX, bMinY, bMaxX, bMaxY := minX-bufferX, minY-bufferY, maxX+bufferX, maxY+bufferY

	sqlColumns := maplayer.ConstructSQLColumns(layer, true)

	var rawSQL string

	// Check if this is a Point layer and should use clustering
	if layer.GeometryType == "Point" && z < 16 {
		// Clustering SQL for Point layers
		rawSQL = `
		SELECT ST_AsMVT(tile, ?, ?, ?) FROM (
			SELECT
				CASE
					WHEN cluster_id IS NOT NULL THEN
						jsonb_build_object(
							'cluster', true,
							'point_count', count(*)::int,
							'item_ids', to_jsonb((array_agg(` + layer.IDFieldName + `))[1:50])::text,
							'point_count_abbreviated',
								CASE
									WHEN count(*) >= 10000 THEN (count(*)/1000)::int || 'K'
									WHEN count(*) >= 1000 THEN ROUND(count(*)/1000.0, 1)::text || 'K'
									ELSE count(*)::text
								END
						)
					ELSE
						to_jsonb((array_agg(points))[1]) - '` + layer.GeometryFieldName + `' - 'cluster_id'
				END as properties,
				ST_AsMVTGeom(
						ST_Centroid(ST_Collect(` + layer.GeometryFieldName + `)),
					ST_MakeEnvelope(?, ?, ?, ?, 4326),
					?,
					?,
					true
				) AS ` + layer.GeometryFieldName + `
			FROM (
				SELECT
					*,
					ST_ClusterDBSCAN(
						` + layer.GeometryFieldName + `,
						eps := ?,
						minpoints := 2
					) OVER () as cluster_id
				FROM ` + layer.DbSchema + `.` + layer.DbTable + `
				WHERE ` + layer.GeometryFieldName + ` && ST_MakeEnvelope(?, ?, ?, ?, 4326) %s
			) points
			GROUP BY
				cluster_id,
				CASE WHEN cluster_id IS NULL THEN ` + layer.IDFieldName + ` ELSE NULL END
		) AS tile
		WHERE ` + layer.GeometryFieldName + ` IS NOT NULL
		`
	} else {
		// Standard SQL for non-Point layers or high zoom levels
		rawSQL = `
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
	}

	userMap, _ := user.(map[string]interface{})

	query := rawSQL
	var filterConditions []string
	var filterValues []interface{}

	if layer.IsPermission {
		if len(layer.RolePermissions) > 0 {
			roleVal, ok := userMap["role"]
			roleFloat, isFloat := roleVal.(float64)
			roleInt := int(roleFloat)
			if !ok || !isFloat {
				return nil, errors.New("user role is missing or not a float")
			}

			hasPermission := false
			roleFound := false

			for _, perm := range layer.RolePermissions {
				if roleInt == perm.RoleID {
					roleFound = true
					break
				}
			}

			if layer.IsRoleException != nil && *layer.IsRoleException != 0 {
				hasPermission = !roleFound
			} else {
				hasPermission = roleFound
			}

			if !hasPermission {
				return nil, errors.New("user role does not have permission for this layer")
			}
		}

		if len(layer.UserPermissions) > 0 {
			idVal, ok := userMap["id"]
			idInt64, isInt64 := idVal.(int64)
			if !ok || !isInt64 {
				return nil, fmt.Errorf("user id is missing or not an int64")
			}

			hasPermission := false
			userFound := false

			for _, perm := range layer.UserPermissions {
				if idInt64 == int64(perm.UserID) {
					userFound = true
					break
				}
			}

			if layer.IsRoleException != nil && *layer.IsRoleException != 0 {
				hasPermission = !userFound
			} else {
				hasPermission = userFound
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

	for key, value := range areaFilters {
		if key == "districtID" && layer.SoumIDField != nil && *layer.SoumIDField != "" {
			filterConditions = append(filterConditions, fmt.Sprintf("AND %s = ?", *layer.SoumIDField))
			filterValues = append(filterValues, value)
		} else if key == "regionID" && layer.BaghIDField != nil && *layer.BaghIDField != "" {
			filterConditions = append(filterConditions, fmt.Sprintf("AND %s = ?", *layer.BaghIDField))
			filterValues = append(filterValues, value)
		}
	}

	query = fmt.Sprintf(query, strings.Join(filterConditions, " "))

	var args []interface{}

	// Build args based on whether clustering is enabled
	if layer.GeometryType == "Point" && z < 16 {
		// Clustering query args
		// Use center latitude for radius calculation to reduce distortion
		centerLat := (minY + maxY) / 2.0
		clusterRadius := getClusterRadius(z, centerLat)

		args = []interface{}{
			layer.DbSchema + "." + layer.DbTable,
			tileSize,
			layer.GeometryFieldName,
			minX, minY, maxX, maxY, // ST_MakeEnvelope (Clip) uses standard bbox
			tileSize,
			tileExtent,
			clusterRadius, // eps parameter for ST_ClusterDBSCAN
			// Use BUFFERED bbox for selection to include edge points
			bMinX, bMinY, bMaxX, bMaxY,
		}
	} else {
		// Standard query args
		args = []interface{}{
			layer.DbSchema + "." + layer.DbTable,
			tileSize,
			layer.GeometryFieldName,
			minX, minY, maxX, maxY,
			tileSize,
			tileExtent,
			// Use BUFFERED bbox for standard queries too to be safe (optional but recommended)
			// But for now, let's stick to standard behavior unless requested
			// Wait, MVT usually benefits from buffering too.
			// Let's keep using buffered bbox for selection.
			bMinX, bMinY, bMaxX, bMaxY,
		}
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

	return tileHandler(layerDetails, nil, nil, nil)(c)
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
	areaFilters := make(map[string]string)

	for key, value := range query {
		if key == "districtID" || key == "regionID" {
			areaFilters[key] = value
		} else {
			filters[key] = value
		}
	}

	layerDetails, err := maplayer.FetchLayerDetails(layer)
	if err != nil {
		log.Printf("Layer not found: %v", err)
		return c.Status(fiber.StatusNotFound).SendString("Layer not found")
	}

	return tileHandler(layerDetails, nil, filters, areaFilters)(c)
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
	areaFilters := make(map[string]string)

	for key, value := range query {
		if key == "districtID" || key == "regionID" {
			areaFilters[key] = value
		} else {
			filters[key] = value
		}
	}

	layerDetails, err := maplayer.FetchLayerDetails(layer)
	if err != nil {
		log.Printf("Layer not found: %v", err)
		return c.Status(fiber.StatusNotFound).SendString("Layer not found")
	}

	return tileHandler(layerDetails, user, filters, areaFilters)(c)
}
