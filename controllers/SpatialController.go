package controllers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/khankhulgun/khanmap/maplayer"
	"github.com/khankhulgun/khanmap/models"
	"github.com/khankhulgun/khanmap/spatial"
)

func Spatial(c *fiber.Ctx) error {
	layer := c.Params("layer")
	relationship := c.Params("relationship")

	if layer == "" || relationship == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Layer and relationship are required parameters",
		})
	}

	// Parse the request body into the GeometryInput struct
	var input models.GeometryInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Invalid input",
			"error":   err.Error(),
		})
	}

	// Validate the input to ensure geometry is provided
	if input.Geometry == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Geometry is required",
		})
	}

	// Fetch layer details
	layerDetails, err := maplayer.FetchLayerDetails(layer)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":  "error",
			"message": "Layer not found",
		})
	}

	// Get the corresponding PostGIS function for the relationship
	sqlFunction, err := spatial.GetRelationshipFunction(relationship)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": err.Error(),
		})
	}

	// Adjust the columns to select based on input
	if input.OutFields != "*" && input.OutFields != "" {
		layerDetails.ColumnSelects = layerDetails.IDFieldName + "," + input.OutFields
	} else if input.OutFields == "" {
		layerDetails.ColumnSelects = layerDetails.IDFieldName
	}

	// Build and execute the spatial query
	query := spatial.BuildSpatialQuery(layerDetails, sqlFunction, input.Geometry, input.ReturnGeometry)
	results, err := spatial.ExecuteSpatialQuery(query, input.Geometry)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Error executing spatial query",
			"error":   err.Error(),
		})
	}

	return c.JSON(results)
}
