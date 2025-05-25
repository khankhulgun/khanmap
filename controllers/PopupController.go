package controllers

import (
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/khankhulgun/khanmap/maplayer"
	"github.com/khankhulgun/khanmap/models"
	"github.com/khankhulgun/khanmap/spatial"
	"github.com/lambda-platform/lambda/DB"
	"strings"
)

func GetMapData(c *fiber.Ctx) error {
	// Parse input JSON
	var input struct {
		Geometry string   `json:"geometry"`
		Layers   []string `json:"layers"`
	}

	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Invalid input",
			"error":   err.Error(),
		})
	}
	if input.Geometry == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Geometry is required",
		})
	}

	// Additional check to ensure input.Geometry is valid
	if !strings.HasPrefix(input.Geometry, "POINT(") && !strings.HasPrefix(input.Geometry, "LINESTRING(") && !strings.HasPrefix(input.Geometry, "POLYGON(") {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Invalid geometry format",
		})
	}

	// Buffer size in meters for Point and LineString geometries (adjust as needed)
	const bufferSize = 60.0 // Example buffer size in meters

	// Prepare results container with group structure
	groupedResults := make(map[string]struct {
		LayerName string                   `json:"layer_name"`
		Features  []map[string]interface{} `json:"features"`
	})

	// Loop through each layer ID and perform a query
	for _, layerID := range input.Layers {
		// Fetch layer details
		layerDetails, err := maplayer.FetchLayerDetails(layerID)
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"status":  "error",
				"message": fmt.Sprintf("Layer %s not found", layerID),
			})
		}

		// Determine if buffering is needed for precision
		queryGeometry := input.Geometry
		if layerDetails.GeometryType == "LineString" || layerDetails.GeometryType == "Point" {
			// Apply buffering to the input geometry for better spatial matching
			queryGeometry = fmt.Sprintf("ST_Buffer(ST_GeomFromText('%s', 4326)::geography, %f)::geometry", input.Geometry, bufferSize)
		}

		sqlFunction := "ST_Intersects" // Example spatial function
		query := spatial.BuildSpatialQuery(layerDetails, sqlFunction, queryGeometry, false)

		if layerDetails.GeometryType == "LineString" || layerDetails.GeometryType == "Point" {
			query = spatial.BuildSpatialQueryWithFromText(layerDetails, sqlFunction, queryGeometry, false)

			queryGeometry = ""
		}
		// Execute the query
		layerResults, err := spatial.ExecuteSpatialQuery(query, queryGeometry)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"status":  "error",
				"message": "Error executing spatial query",
				"error":   err.Error(),
			})
		}

		// Group results by layerID
		if _, exists := groupedResults[layerID]; !exists {
			groupedResults[layerID] = struct {
				LayerName string                   `json:"layer_name"`
				Features  []map[string]interface{} `json:"features"`
			}{
				LayerName: layerDetails.LayerTitle, // Assuming LayerTitle contains the layer's name
				Features:  []map[string]interface{}{},
			}
		}

		// Append the features to the appropriate group
		group := groupedResults[layerID]
		group.Features = append(group.Features, layerResults...)
		groupedResults[layerID] = group
	}

	// Convert map to a slice of grouped results for output
	var output []map[string]interface{}
	for layerID, group := range groupedResults {
		if len(group.Features) >= 1 {
			output = append(output, map[string]interface{}{
				"layer_id":   layerID,
				"layer_name": group.LayerName,
				"features":   group.Features,
			})
		}
	}

	// Return the grouped results
	return c.JSON(output)
}

func FilterOptions(c *fiber.Ctx) error {
	filterIDs := c.Query("ids")
	if filterIDs == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "IDs query param is required",
		})
	}

	idStrings := strings.Split(filterIDs, ",")
	var filters []models.MapFilters
	if err := DB.DB.Where("id IN ?", idStrings).Find(&filters).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Error retrieving map filters",
			"error":   err.Error(),
		})
	}

	type GroupedResult struct {
		Label        string                   `json:"label"`
		FilterID     string                   `json:"filter_id"`
		ParentID     string                   `json:"parent_id"`
		ParentColumn string                   `json:"parent_column"`
		ValueField   string                   `json:"value_field"`
		LabelField   string                   `json:"label_field"`
		FilterOrder  int                      `json:"filter_order"`
		Options      []map[string]interface{} `json:"options"`
	}

	response := make([]GroupedResult, 0, len(filters))
	for _, filter := range filters {
		columns := []string{"id"}
		if filter.ValueField != "id" {
			columns = append(columns, filter.ValueField)
		}
		for _, col := range strings.Split(filter.LabelField, ",") {
			columns = append(columns, strings.TrimSpace(col))
		}
		if filter.ParentFilterInTable != nil && *filter.ParentFilterInTable != "" {
			if !contains(columns, *filter.ParentFilterInTable) {
				columns = append(columns, *filter.ParentFilterInTable)
			}
		}

		var results []map[string]interface{}
		if err := DB.DB.Select(strings.Join(columns, ",")).Table(filter.Schema + "." + filter.Table).Find(&results).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"status":  "error",
				"message": "Error retrieving map filters",
				"filter":  filter.ID,
				"error":   err.Error(),
			})
		}

		response = append(response, GroupedResult{
			FilterID:     filter.ID,
			Label:        filter.Label,
			ParentID:     ptrString(filter.ParentFilterID),
			ParentColumn: ptrString(filter.ParentFilterInTable),
			ValueField:   filter.ValueField,
			LabelField:   filter.LabelField,
			FilterOrder:  ptrInt(filter.FilterOrder),
			Options:      results,
		})
	}

	return c.JSON(response)
}

// Helper functions
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func ptrString(ptr *string) string {
	if ptr == nil {
		return ""
	}
	return *ptr
}

func ptrInt(ptr *int) int {
	if ptr == nil {
		return 0
	}
	return *ptr
}
