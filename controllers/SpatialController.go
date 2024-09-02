package controllers

import (
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/khankhulgun/khanmap/models"
	"github.com/khankhulgun/khanmap/tiles"
	"github.com/lambda-platform/lambda/DB"
	"strings"
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
	layerDetails, err := tiles.FetchLayerDetails(layer)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":  "error",
			"message": "Layer not found",
		})
	}

	// Define the SQL relationship function mapping
	relationshipFunctions := map[string]string{
		"contains":   "ST_Contains",
		"crosses":    "ST_Crosses",
		"disjoint":   "ST_Disjoint",
		"equals":     "ST_Equals",
		"intersects": "ST_Intersects",
		"overlaps":   "ST_Overlaps",
		"within":     "ST_Within",
		"touches":    "ST_Touches",
	}

	// Get the corresponding PostGIS function for the relationship
	sqlFunction, ok := relationshipFunctions[strings.ToLower(relationship)]
	if !ok {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Invalid spatial relationship",
		})
	}

	if input.OutFields != "*" && input.OutFields != "" {
		layerDetails.ColumnSelects = layerDetails.IDFieldName + "," + input.OutFields
	} else if input.OutFields == "" {
		layerDetails.ColumnSelects = layerDetails.IDFieldName
	}

	if input.ReturnGeometry {
		layerDetails.ColumnSelects = layerDetails.ColumnSelects + "," + layerDetails.GeometryFieldName
	}

	// Construct the SQL query
	query := fmt.Sprintf(`
		SELECT %s FROM %s.%s
		WHERE %s(%s, ST_GeomFromText(?, 4326))
	`, tiles.ConstructSQLColumns(layerDetails, false), layerDetails.DbSchema, layerDetails.DbTable, sqlFunction, layerDetails.GeometryFieldName)

	// Execute the query
	var results []map[string]interface{}
	if err := DB.DB.Raw(query, input.Geometry).Scan(&results).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Error executing spatial query",
			"error":   err.Error(),
		})
	}

	return c.JSON(results)
}
