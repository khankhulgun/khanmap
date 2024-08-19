package controllers

import (
	"errors"
	"github.com/gofiber/fiber/v2"
	"github.com/khankhulgun/khanmap/models"
	"github.com/lambda-platform/lambda/DB"
	"gorm.io/gorm"
)

func GetMapLayers(c *fiber.Ctx) error {

	// Get the 'id' parameter from the URL
	id := c.Params("id")
	if id == "" {
		// Return a 400 Bad Request error if no ID is provided
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "ID parameter is required",
		})
	}

	// Declare a variable to store the map layer
	var currentMap models.Map

	// Adjust preloading to correctly apply ordering to Categories and Layers
	result := DB.DB.Preload("Categories", func(db *gorm.DB) *gorm.DB {
		return db.Order("category_order ASC").Where("is_active = ?", true).Preload("Layers", func(db *gorm.DB) *gorm.DB {
			return db.Order("layer_order ASC").Where("is_active = ?", true).Preload("Legends")
		})
	}).Where("id = ?", id).First(&currentMap)

	// Check for errors in the query, such as "Record Not Found"
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			// Return a 404 Not Found error if the map layer is not found
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"status":  "error",
				"message": "Map layer not found",
			})
		}
		// Return a 500 Internal Server Error for any other database errors
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Error retrieving map layer",
			"error":   result.Error,
		})
	}

	// Return the map layer as JSON
	return c.JSON(currentMap)
}
