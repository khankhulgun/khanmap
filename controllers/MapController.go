package controllers

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/khankhulgun/khanmap/models"
	"github.com/khankhulgun/khanmap/sprite"
	"github.com/lambda-platform/lambda/DB"
	"github.com/lambda-platform/lambda/config"
	"gorm.io/gorm"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func GetMapLayers(c *fiber.Ctx) error {

	// Get the 'id' parameter from the URL
	id := c.Params("id")
	generate := c.Query("generate")
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
			return db.Order("layer_order ASC").Where("is_active = ?", true).
				Preload("Legends", func(db *gorm.DB) *gorm.DB {
					return db.Order("legend_order ASC")
				}).
				Preload("Permissions")
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

	mapStyle, generateErr := generateVectorTileStyle(currentMap.Categories)

	currentMap.Version = mapStyle.Version
	currentMap.Layers = mapStyle.Layers
	currentMap.Sources = mapStyle.Sources

	spriteURL := config.LambdaConfig.Domain + "/map/" + id + "/sprite/" + id

	// Check if the sprite URL already starts with http:// or https://
	hasProtocol := strings.HasPrefix(spriteURL, "http://") || strings.HasPrefix(spriteURL, "https://")

	if !hasProtocol {
		// If no protocol, prepend https://
		currentMap.Sprite = "https://" + spriteURL
	} else {
		currentMap.Sprite = spriteURL
	}

	if generate == "true" {
		if generateErr != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"status": "error",
				"error":  generateErr.Error(),
			})
		}
		// Create the JSON output path
		outputDir := "./public/map"
		err := os.MkdirAll(outputDir, os.ModePerm) // Ensure the directory exists
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"status":  "error",
				"message": "Error creating output directory",
				"error":   err.Error(),
			})
		}

		// Define the output file path
		outputFile := filepath.Join(outputDir, fmt.Sprintf("%s.json", id))

		// Serialize the currentMap object to JSON
		file, err := os.Create(outputFile)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"status":  "error",
				"message": "Error creating JSON file",
				"error":   err.Error(),
			})
		}
		defer file.Close()

		encoder := json.NewEncoder(file)
		encoder.SetIndent("", "  ") // Pretty-print JSON if desired
		if err := encoder.Encode(currentMap); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"status":  "error",
				"message": "Error writing JSON file",
				"error":   err.Error(),
			})
		}

		sprite.MakeSprite(fmt.Sprintf("./public/map/%s/sprite/images", id), fmt.Sprintf("./public/map/%s/sprite/%s", id, id))
	}

	return c.JSON(currentMap)
}

func generateVectorTileStyle(categories []models.ViewMapLayerCategories) (models.VectorTileStyle, error) {
	var style models.VectorTileStyle

	// Initialize sources with error handling
	style.Sources = map[string]models.VectorSource{}

	for _, category := range categories {

		for _, layer := range category.Layers {

			baseUrl := config.LambdaConfig.Domain
			hasProtocol := strings.HasPrefix(baseUrl, "http://") || strings.HasPrefix(baseUrl, "https://")

			if !hasProtocol {
				// If no protocol, prepend https://
				baseUrl = "https://" + baseUrl
			}
			style.Sources[layer.ID] = models.VectorSource{

				Type:  "vector",
				Tiles: []string{baseUrl + "/tiles/" + layer.ID + "/{z}/{x}/{y}.pbf"},
			}
		}

	}

	// Iterate through categories and layers, defining styles based on geometry type
	for _, category := range categories {
		for _, layer := range category.Layers {
			switch layer.GeometryType {
			case "Point":

				// Define line layer style using line color, width, and other properties
				if len(layer.Legends) >= 1 {
					if layer.Legends[0].Marker != nil {

						pointSymbol := models.SymbolLayer{
							ID:          layer.ID,
							Type:        "symbol",
							Source:      layer.ID, // Use category source
							SourceLayer: layer.DbSchema + "." + layer.DbTable,
							Layout: models.SymbolLayerLayout{
								IconImage:           layer.ID,
								IconSize:            1.0,
								IconAllowOverlap:    true,
								IconIgnorePlacement: false,
								IconOffset:          []int{0, 0},
							},
							Paint: models.SymbolLayerPaint{
								IconColor: "#000000",
							},
						}
						style.Layers = append(style.Layers, pointSymbol)

						// Define the output directory
						outputDir := fmt.Sprintf("./public/map/%s/sprite/images", category.MapID)
						err := os.MkdirAll(outputDir, os.ModePerm) // Ensure directory exists
						if err != nil {
							return style, errors.New("Error creating output directory")
						}

						// Assuming `markerPath` contains the path to the marker file (e.g., "path/to/marker.svg" or "path/to/marker.png")
						markerPath := *layer.Legends[0].Marker                      // Example: Assuming `layer.Marker` contains the file path to the marker
						outputFile := fmt.Sprintf("%s/%s.png", outputDir, layer.ID) // Define output file name

						// Check if the marker is an SVG
						if strings.HasSuffix(markerPath, ".svg") {
							// Convert SVG to PNG

							err = sprite.SVGToPNG("./public"+markerPath, outputFile)
							if err != nil {
								return style, errors.New("Error converting SVG to PNG")
							}
						} else if strings.HasSuffix(markerPath, ".png") {
							// Copy the PNG marker to the target location
							input, err := os.Open(markerPath)
							if err != nil {
								return style, errors.New("Error opening marker file")
							}
							defer input.Close()

							output, err := os.Create(outputFile)
							if err != nil {
								return style, errors.New("Error creating output file")
							}
							defer output.Close()

							_, err = io.Copy(output, input)
							if err != nil {
								return style, errors.New("Error copying marker file")
							}
						} else {
							return style, errors.New("Unsupported marker file format")
						}
					}
				}
			case "LineString":
				// Define line layer style using line color, width, and other properties
				if len(layer.Legends) >= 1 {
					if layer.Legends[0].FillColor != nil {
						lineLayer := models.LineLayer{
							ID:          layer.ID,
							Type:        "line",
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
							Type:        "fill",
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
							Type:        "line",
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
