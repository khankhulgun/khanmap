package controllers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/khankhulgun/khanmap/models"
	"github.com/khankhulgun/khanmap/sprite"
	"github.com/lambda-platform/lambda/DB"
	agentUtils "github.com/lambda-platform/lambda/agent/utils"
	"github.com/lambda-platform/lambda/config"
	"gorm.io/gorm"
)

func GetMapLayers(c *fiber.Ctx) error {
	id := c.Params("id")
	generate := c.Query("generate")
	secure := c.Query("secure")

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
	result := DB.DB.Preload("Filters").Preload("Categories", func(db *gorm.DB) *gorm.DB {
		return db.Order("category_order ASC").Where("is_active = ?", true).
			Preload("Layers", func(db *gorm.DB) *gorm.DB {
				return db.Order("layer_order ASC").Where("is_public = ? AND is_active = ?", true, true).
					Preload("Legends", func(db *gorm.DB) *gorm.DB {
						return db.Order("legend_order ASC")
					}).
					Preload("AdminFilters")
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

	var filteredCategories []models.ViewMapLayerCategories
	for _, category := range currentMap.Categories {
		if len(category.Layers) > 0 {
			filteredCategories = append(filteredCategories, category)
		}
	}
	currentMap.Categories = filteredCategories

	mapStyle, generateErr := generateVectorTileStyle(currentMap.Categories, secure)

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

func GetMapLayersWithAuth(c *fiber.Ctx) error {
	id := c.Params("id")
	secure := c.Query("secure")
	if id == "" {
		// Return a 400 Bad Request error if no ID is provided
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "ID parameter is required",
		})
	}

	user, err := agentUtils.AuthUserObject(c)
	if err != nil {
		log.Printf("User not found: %v", err)
		return c.Status(fiber.StatusUnauthorized).SendString("User not found")
	}

	roleVal, ok := user["role"]
	roleFloat, isFloat := roleVal.(float64)
	roleInt := int(roleFloat)
	if !ok || !isFloat {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status":  "error",
			"message": "User role is missing or not a float",
		})
	}

	idVal, ok := user["id"]
	idInt64, isInt64 := idVal.(int64)
	if !ok || !isInt64 {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status":  "error",
			"message": "User id is missing or not an int64",
		})
	}

	var currentMap models.Map
	result := DB.DB.Preload("Filters").Preload("Categories", func(db *gorm.DB) *gorm.DB {
		return db.Order("category_order ASC").Where("is_active = ?", true).
			Preload("Layers", func(db *gorm.DB) *gorm.DB {
				return db.Order("layer_order ASC").Where("is_active = ?", true).
					Preload("Legends", func(db *gorm.DB) *gorm.DB {
						return db.Order("legend_order ASC")
					}).
					Preload("AdminFilters").
					Preload("RolePermissions").
					Preload("UserPermissions")
			})
	}).Where("id = ?", id).First(&currentMap)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"status":  "error",
				"message": "Map layer not found",
			})
		}

		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Error retrieving map layer",
			"error":   result.Error,
		})
	}

	var filteredCategories []models.ViewMapLayerCategories

	for i := range currentMap.Categories {
		var filteredLayers []models.MapLayers

		for _, layer := range currentMap.Categories[i].Layers {
			shouldIncludeLayer := true

			if layer.IsPermission {
				if len(layer.RolePermissions) > 0 {
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
						shouldIncludeLayer = false
					}
				}

				if len(layer.UserPermissions) > 0 {
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
						shouldIncludeLayer = false
					}
				}
			}

			if shouldIncludeLayer {
				filteredLayers = append(filteredLayers, layer)
			}
		}

		if len(filteredLayers) > 0 {
			currentMap.Categories[i].Layers = filteredLayers
			filteredCategories = append(filteredCategories, currentMap.Categories[i])
		}
	}

	currentMap.Categories = filteredCategories

	for i := range currentMap.Categories {
		for j := range currentMap.Categories[i].Layers {
			currentMap.Categories[i].Layers[j].RolePermissions = nil
			currentMap.Categories[i].Layers[j].UserPermissions = nil
		}
	}

	mapStyle, generateErr := generateVectorTileStyle(currentMap.Categories, secure)

	currentMap.Version = mapStyle.Version
	currentMap.Layers = mapStyle.Layers
	currentMap.Sources = mapStyle.Sources

	spriteURL := config.LambdaConfig.Domain + "/map/" + id + "/sprite/" + id
	hasProtocol := strings.HasPrefix(spriteURL, "http://") || strings.HasPrefix(spriteURL, "https://")

	if !hasProtocol {
		currentMap.Sprite = "https://" + spriteURL
	} else {
		currentMap.Sprite = spriteURL
	}

	if generateErr != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status": "error",
			"error":  generateErr.Error(),
		})
	}

	return c.JSON(currentMap)
}

func generateVectorTileStyle(categories []models.ViewMapLayerCategories, secure string) (models.VectorTileStyle, error) {
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
			var tilePath string = "/tiles/"

			if secure == "true" {
				tilePath = "/tiles-with-permission/"
			}
			style.Sources[layer.ID] = models.VectorSource{

				Type:  "vector",
				Tiles: []string{baseUrl + tilePath + layer.ID + "/{z}/{x}/{y}.pbf"},
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
								return style, errors.New("Error converting SVG to PNG: " + layer.LayerTitle + layer.ID)
							}
						} else if strings.HasSuffix(markerPath, ".png") {
							// Copy the PNG marker to the target location
							input, err := os.Open(markerPath)
							if err != nil {
								return style, errors.New("Error opening marker file: " + layer.LayerTitle + layer.ID)
							}
							defer input.Close()

							output, err := os.Create(outputFile)
							if err != nil {
								return style, errors.New("Error creating output file: " + layer.LayerTitle + layer.ID)
							}
							defer output.Close()

							_, err = io.Copy(output, input)
							if err != nil {
								return style, errors.New("Error copying marker file: " + layer.LayerTitle + layer.ID)
							}
						} else {
							return style, errors.New("Unsupported marker file format: " + layer.LayerTitle + layer.ID)
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
