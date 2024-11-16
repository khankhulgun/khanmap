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
			return db.Order("layer_order ASC").Where("is_active = ?", true).Preload("Legends").Preload("Permissions")
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

	mapStyle, _ := generateVectorTileStyle(currentMap.Categories)

	currentMap.Version = mapStyle.Version
	currentMap.Layers = mapStyle.Layers
	currentMap.Sources = mapStyle.Sources
	currentMap.Sprite = mapStyle.Sprite
	return c.JSON(currentMap)
}

func generateVectorTileStyle(categories []models.ViewMapLayerCategories) (models.VectorTileStyle, error) {
	var style models.VectorTileStyle

	// Initialize sources with error handling
	style.Sources = map[string]models.VectorSource{}
	for _, category := range categories {

		for _, layer := range category.Layers {

			style.Sources[layer.IDFieldname] = models.VectorSource{
				Type:  "vector",
				Tiles: []string{"https://riskmapping.mn/tiles/" + layer.ID + "/{z}/{x}/{y}.pbf"},
			}
		}

	}

	// Iterate through categories and layers, defining styles based on geometry type
	for _, category := range categories {
		for _, layer := range category.Layers {
			switch layer.GeometryType {
			case "Point":
				// Define point layer style using category icon and other properties
				//pointSymbol := models.PointLayerSymbol{
				//	ID:          layer.ID,
				//	Source:      category.IDFieldName, // Use category source
				//	SourceLayer: layer.IDFieldName,
				//	Layout: models.PointLayerSymbolLayout{
				//		IconImage:           category.Icon,
				//		IconSize:            1.0,
				//		IconAllowOverlap:    true,
				//		IconIgnorePlacement: true,
				//	},
				//	Paint: models.PointLayerSymbolPaint{
				//		IconColor: "#000000", // Replace with desired color
				//	},
				//}
				//style.PointLayerSymbols = append(style.PointLayerSymbols, pointSymbol)
			case "LineString":
				// Define line layer style using line color, width, and other properties
				if len(layer.Legends) >= 1 {
					if layer.Legends[0].FillColor != nil {
						lineLayer := models.LineLayer{
							ID:          layer.ID,
							Source:      layer.ID, // Use category source
							SourceLayer: layer.DbSchema + "." + layer.DbTable,
							Paint: models.LineLayerPaint{
								LineColor: *layer.Legends[0].FillColor,
								LineWidth: 2.0,
							},
						}
						style.Layers = append(style.Layers, lineLayer)
					}

				}

			case "Polygon":
				if len(layer.Legends) >= 1 {
					if layer.Legends[0].FillColor != nil && layer.Legends[0].StrokeColor != nil {
						fillLayer := models.FillLayer{
							ID:          layer.ID,
							Source:      layer.ID, // Use category source
							SourceLayer: layer.DbSchema + "." + layer.DbTable,
							Paint: models.FillLayerPaint{
								FillColor:   *layer.Legends[0].FillColor,
								FillOpacity: 0.6,
							},
						}
						style.Layers = append(style.Layers, fillLayer)

						lineLayer := models.LineLayer{
							ID:          layer.ID,
							Source:      layer.ID, // Use category source
							SourceLayer: layer.DbSchema + "." + layer.DbTable,
							Paint: models.LineLayerPaint{
								LineColor: *layer.Legends[0].StrokeColor,
								LineWidth: 2.0,
							},
						}
						style.Layers = append(style.Layers, lineLayer)
					}
				}
			}
		}
	}

	return style, nil
}
