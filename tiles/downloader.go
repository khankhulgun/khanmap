package tiles

import (
	"fmt"
	"github.com/khankhulgun/khanmap/models"
	"github.com/lambda-platform/lambda/DB"
	"math"
	"os"
	"path/filepath"
)

// Constants for download directory and zoom levels
const (
	downloadDir = "./public/saved-tiles" // Directory to save tiles
	minZoom     = 0                      // Minimum zoom level
	maxZoom     = 18                     // Maximum zoom level
)

// BoundingBox represents the spatial extent for a layer
type BoundingBox struct {
	MinLat float64 `gorm:"column:min_lat"`
	MaxLat float64 `gorm:"column:max_lat"`
	MinLon float64 `gorm:"column:min_lon"`
	MaxLon float64 `gorm:"column:max_lon"`
}

// GetBoundingBox fetches the bounding box from the database for a given layerID using GORM
func GetBoundingBox(layer models.MapLayersForTile) (*BoundingBox, error) {
	var bbox BoundingBox

	// Construct the query to get the bounding box for the entire layer as a single result
	query := fmt.Sprintf(`
		SELECT 
			ST_YMin(ST_Extent(%s)) as min_lat, 
			ST_YMax(ST_Extent(%s)) as max_lat, 
			ST_XMin(ST_Extent(%s)) as min_lon, 
			ST_XMax(ST_Extent(%s)) as max_lon 
		FROM %s.%s`,
		layer.GeometryFieldName,
		layer.GeometryFieldName,
		layer.GeometryFieldName,
		layer.GeometryFieldName,
		layer.DbSchema,
		layer.DbTable)

	// Execute the query using GORM's raw SQL handling
	if err := DB.DB.Raw(query).Scan(&bbox).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch bounding box: %w", err)
	}

	return &bbox, nil
}

// Clamps the latitude to the valid bounds of the Mercator projection
func clampLatitude(lat float64) float64 {
	if lat > 85.0511287798 {
		return 85.0511287798
	}
	if lat < -85.0511287798 {
		return -85.0511287798
	}
	return lat
}

// latLonToTileXY converts latitude and longitude to tile coordinates at a given zoom level
func latLonToTileXY(lat, lon float64, zoom int) (x, y int) {
	// Clamp latitude to prevent invalid tile ranges
	lat = clampLatitude(lat)

	latRad := lat * math.Pi / 180.0 // Convert latitude to radians
	n := math.Pow(2, float64(zoom)) // Number of tiles per axis at this zoom level

	// Calculate the tile coordinates
	x = int((lon + 180.0) / 360.0 * n)
	y = int((1.0 - math.Log(math.Tan(latRad)+1.0/math.Cos(latRad))/math.Pi) / 2.0 * n)

	return x, y
}

// Ensure min/max tile Y values are ordered correctly
func ensureValidTileRange(minTileX, maxTileX, minTileY, maxTileY *int) {
	// Swap X values if inverted
	if *minTileX > *maxTileX {
		*minTileX, *maxTileX = *maxTileX, *minTileX
	}
	// Swap Y values if inverted
	if *minTileY > *maxTileY {
		*minTileY, *maxTileY = *maxTileY, *minTileY
	}
}

// CreateTiles downloads all tiles for a given layerID and zoom levels
func CreateTiles(layer models.MapLayersForTile) error {
	// Fetch the bounding box from the database
	bbox, err := GetBoundingBox(layer)
	if err != nil {
		return err
	}

	// Log the bounding box for debugging
	fmt.Printf("Bounding Box: MinLat: %f, MaxLat: %f, MinLon: %f, MaxLon: %f\n", bbox.MinLat, bbox.MaxLat, bbox.MinLon, bbox.MaxLon)

	// Loop through each zoom level
	for zoom := minZoom; zoom <= maxZoom; zoom++ {
		fmt.Println("Processing zoom level:", zoom)

		// Get tile ranges for the bounding box
		minTileX, minTileY := latLonToTileXY(bbox.MinLat, bbox.MinLon, zoom)
		maxTileX, maxTileY := latLonToTileXY(bbox.MaxLat, bbox.MaxLon, zoom)

		// Ensure the tile ranges are valid (swap if needed)
		ensureValidTileRange(&minTileX, &maxTileX, &minTileY, &maxTileY)

		// Log tile range for debugging
		fmt.Printf("Zoom level %d: minTileX=%d, maxTileX=%d, minTileY=%d, maxTileY=%d\n", zoom, minTileX, maxTileX, minTileY, maxTileY)

		if minTileX > maxTileX || minTileY > maxTileY {
			fmt.Printf("Skipping zoom level %d due to invalid tile range.\n", zoom)
			continue
		}

		// Iterate over the tiles within the bounding box
		for x := minTileX; x <= maxTileX; x++ {
			for y := minTileY; y <= maxTileY; y++ {
				// Construct the local file path
				filePath := filepath.Join(downloadDir, fmt.Sprintf("%s/%d/%d/%d.pbf", layer.ID, zoom, x, y))

				// Create directories if they don't exist
				err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm)
				if err != nil {
					fmt.Printf("Failed to create directories: %v\n", err)
					continue
				}

				mvtData, err := getVectorTile(zoom, x, y, layer, nil)
				if err != nil {
					fmt.Printf("Failed to get vector tile for zoom %d, x=%d, y=%d: %v\n", zoom, x, y, err)
					continue
				}

				// Create the file
				file, err := os.Create(filePath)
				if err != nil {
					return fmt.Errorf("failed to create file: %w", err)
				}
				defer file.Close()

				// Copy the tile data to the file
				_, err = file.Write(mvtData)
				if err != nil {
					return fmt.Errorf("failed to save tile: %w", err)
				}
			}
		}
	}

	return nil
}
