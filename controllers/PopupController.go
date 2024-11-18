package controllers

import (
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/khankhulgun/khanmap/maplayer"
	"github.com/khankhulgun/khanmap/spatial"
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

	// Prepare results container
	var results []map[string]interface{}

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

		// Construct the spatial query using the location
		sqlFunction := "ST_Intersects" // Example spatial function
		query := spatial.BuildSpatialQuery(layerDetails, sqlFunction, input.Geometry, false)

		// Execute the query
		layerResults, err := spatial.ExecuteSpatialQuery(query, input.Geometry)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"status":  "error",
				"message": "Error executing spatial query",
				"error":   err.Error(),
			})
		}

		// Append results for this layer
		results = append(results, layerResults...)
	}

	// Return the combined results
	return c.JSON(results)
}
