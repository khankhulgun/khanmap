package controllers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/khankhulgun/khanmap/models"
	"github.com/khankhulgun/khanmap/sprite"
	"github.com/lambda-platform/lambda/DB"
	agentUtils "github.com/lambda-platform/lambda/agent/utils"
	"github.com/lambda-platform/lambda/config"
	"gorm.io/gorm"
)

// DoCluster is set globally from khanmap.Set() to enable/disable clustering for all Point layers
var DoCluster bool

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
				return db.Order("layer_order ASC").Where("is_active = ?", true).
					Preload("Legends", func(db *gorm.DB) *gorm.DB {
						return db.Order("legend_order ASC")
					}).
					Preload("AdminFilters").
					Preload("RolePermissions").
					Preload("UserPermissions")
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

	if generate == "true" {
		// Clean old sprite images BEFORE generating new ones
		spriteImagesDir := fmt.Sprintf("./public/map/%s/sprite/images", id)
		oldFiles, _ := filepath.Glob(filepath.Join(spriteImagesDir, "*.png"))
		for _, f := range oldFiles {
			os.Remove(f)
		}
	}

	mapStyle, generateErr := generateVectorTileStyle(currentMap.Categories, secure, generate == "true")

	currentMap.Version = mapStyle.Version
	currentMap.Layers = mapStyle.Layers
	currentMap.Sources = mapStyle.Sources
	currentMap.Glyphs = mapStyle.Glyphs

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

		if spriteErr := sprite.MakeSprite(fmt.Sprintf("./public/map/%s/sprite/images", id), fmt.Sprintf("./public/map/%s/sprite/%s", id, id)); spriteErr != nil {
			log.Printf("MakeSprite error for map %s: %v", id, spriteErr)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"status":  "error",
				"message": "Error generating sprite sheet",
				"error":   spriteErr.Error(),
			})
		}
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
					roleFound := false
					for _, perm := range layer.RolePermissions {
						if roleInt == perm.RoleID {
							roleFound = true
							break
						}
					}

					var hasRolePermission bool
					if layer.IsRoleException != nil && *layer.IsRoleException != 0 {
						hasRolePermission = !roleFound
					} else {
						hasRolePermission = roleFound
					}

					if !hasRolePermission {
						shouldIncludeLayer = false
					}
				}

				if shouldIncludeLayer && len(layer.UserPermissions) > 0 {
					userFound := false
					for _, perm := range layer.UserPermissions {
						if idInt64 == int64(perm.UserID) {
							userFound = true
							break
						}
					}

					var hasUserPermission bool
					if layer.IsRoleException != nil && *layer.IsRoleException != 0 {
						hasUserPermission = !userFound
					} else {
						hasUserPermission = userFound
					}

					if !hasUserPermission {
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

	mapStyle, generateErr := generateVectorTileStyle(currentMap.Categories, secure, false)

	currentMap.Version = mapStyle.Version
	currentMap.Layers = mapStyle.Layers
	currentMap.Sources = mapStyle.Sources
	currentMap.Glyphs = mapStyle.Glyphs

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

func generateVectorTileStyle(categories []models.ViewMapLayerCategories, secure string, generate bool) (models.VectorTileStyle, error) {
	var style models.VectorTileStyle

	// Initialize style properties
	style.Version = 8

	// Set glyphs URL with proper protocol
	baseUrl := config.LambdaConfig.Domain
	if baseUrl == "" {
		baseUrl = "http://localhost:9995"
	}
	hasProtocol := strings.HasPrefix(baseUrl, "http://") || strings.HasPrefix(baseUrl, "https://")
	if !hasProtocol {
		baseUrl = "https://" + baseUrl
	}
	style.Glyphs = baseUrl + "/fonts/{fontstack}/{range}.pbf"

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

				if len(layer.Legends) >= 1 {

					// Check if this layer uses unique value rendering (multiple markers per unique value)
					hasUniqueValueField := layer.UniqueValueField != nil && *layer.UniqueValueField != ""
					hasMultipleUniqueMarkers := false

					if hasUniqueValueField {
						// Check if legends actually have unique_value + marker pairs
						for _, legend := range layer.Legends {
							if legend.UniqueValue != nil && *legend.UniqueValue != "" && legend.Marker != nil && *legend.Marker != "" {
								hasMultipleUniqueMarkers = true
								break
							}
						}
					}

					if hasMultipleUniqueMarkers {
						// === UNIQUE VALUE RENDERING: Multiple markers per unique value ===
						uniqueValueField := *layer.UniqueValueField

						for _, legend := range layer.Legends {
							if legend.UniqueValue == nil || *legend.UniqueValue == "" || legend.Marker == nil || *legend.Marker == "" {
								continue
							}

							uniqueVal := *legend.UniqueValue
							// Create a unique sprite image ID for each unique value
							spriteImageID := layer.ID + "-" + uniqueVal

							// Determine the filter comparison value — use integer if the unique value is numeric
							var filterValue interface{} = uniqueVal
							if intVal, err := strconv.Atoi(uniqueVal); err == nil {
								filterValue = intVal
							}

							// Build filter: unique value match + optionally exclude clustered features
							var symbolFilter []interface{}
							if DoCluster {
								symbolFilter = []interface{}{
									"all",
									[]interface{}{"!", []interface{}{"has", "point_count"}},
									[]interface{}{"==", []interface{}{"get", uniqueValueField}, filterValue},
								}
							} else {
								symbolFilter = []interface{}{
									"==", []interface{}{"get", uniqueValueField}, filterValue,
								}
							}

							pointSymbol := models.SymbolLayer{
								ID:          spriteImageID,
								Type:        "symbol",
								Source:      layer.ID,
								SourceLayer: layer.DbSchema + "." + layer.DbTable,
								Filter:      symbolFilter,
								Layout: models.SymbolLayerLayout{
									IconImage:           spriteImageID,
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

							// Generate sprite image for each unique marker
							if generate {
								outputDir := fmt.Sprintf("./public/map/%s/sprite/images", category.MapID)
								err := os.MkdirAll(outputDir, os.ModePerm)
								if err != nil {
									return style, errors.New("Error creating output directory")
								}

								markerPath := *legend.Marker
								outputFile := fmt.Sprintf("%s/%s.png", outputDir, spriteImageID)

								if strings.HasSuffix(markerPath, ".svg") {
									err = sprite.SVGToPNG("./public"+markerPath, outputFile)
									if err != nil {
										return style, fmt.Errorf("error converting SVG to PNG for layer %s unique value %s (%s), file: %s: %w", layer.LayerTitle, uniqueVal, layer.ID, markerPath, err)
									}
								} else if strings.HasSuffix(markerPath, ".png") {
									input, err := os.Open("./public" + markerPath)
									if err != nil {
										return style, fmt.Errorf("error opening marker file for unique value %s: %s %s", uniqueVal, layer.LayerTitle, layer.ID)
									}
									defer input.Close()

									output, err := os.Create(outputFile)
									if err != nil {
										return style, fmt.Errorf("error creating output file for unique value %s: %s %s", uniqueVal, layer.LayerTitle, layer.ID)
									}
									defer output.Close()

									_, err = io.Copy(output, input)
									if err != nil {
										return style, fmt.Errorf("error copying marker for unique value %s: %s %s", uniqueVal, layer.LayerTitle, layer.ID)
									}
								} else {
									return style, fmt.Errorf("unsupported marker format for unique value %s: %s %s", uniqueVal, layer.LayerTitle, layer.ID)
								}
							}
						}

						// Cluster layers — only if is_cluster is enabled
						if DoCluster {
							// Cluster circles layer (shared for all unique values)
							clusterCircleLayer := models.CircleLayer{
								ID:          layer.ID + "-clusters",
								Type:        "circle",
								Source:      layer.ID,
								SourceLayer: layer.DbSchema + "." + layer.DbTable,
								Filter:      []interface{}{"has", "point_count"},
								Paint: models.CircleLayerPaint{
									CircleColor: []interface{}{
										"step",
										[]interface{}{"get", "point_count"},
										"#05a41b",
										100,
										"#02663a",
										750,
										"#024f34",
									},
									CircleRadius: []interface{}{
										"step",
										[]interface{}{"get", "point_count"},
										20,
										100,
										30,
										750,
										40,
									},
									CircleOpacity:       1,
									CircleStrokeWidth:   5,
									CircleStrokeColor:   "#05a41b",
									CircleStrokeOpacity: 0.4,
								},
							}
							style.Layers = append(style.Layers, clusterCircleLayer)

							// Cluster count text layer (shared)
							clusterCountLayer := models.SymbolLayer{
								ID:          layer.ID + "-cluster-count",
								Type:        "symbol",
								Source:      layer.ID,
								SourceLayer: layer.DbSchema + "." + layer.DbTable,
								Filter:      []interface{}{"has", "point_count"},
								Layout: models.SymbolLayerLayout{
									TextField:  []interface{}{"get", "point_count_abbreviated"},
									TextFont:   []string{"Noto Sans Bold"},
									TextSize:   12,
									TextOffset: []float64{0, 0},
									TextAnchor: "center",
								},
								Paint: models.SymbolLayerPaint{
									TextColor: "#ffffff",
								},
							}
							style.Layers = append(style.Layers, clusterCountLayer)
						}

					} else if layer.Legends[0].Marker != nil {
						// === STANDARD SINGLE MARKER RENDERING ===

						// Build filter: optionally exclude clustered features if clustering is enabled
						var symbolFilter []interface{}
						if DoCluster {
							symbolFilter = []interface{}{"!", []interface{}{"has", "point_count"}}
						}

						pointSymbol := models.SymbolLayer{
							ID:          layer.ID,
							Type:        "symbol",
							Source:      layer.ID,
							SourceLayer: layer.DbSchema + "." + layer.DbTable,
							Filter:      symbolFilter,
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

						// Cluster layers — only if is_cluster is enabled
						if DoCluster {
							clusterCircleLayer := models.CircleLayer{
								ID:          layer.ID + "-clusters",
								Type:        "circle",
								Source:      layer.ID,
								SourceLayer: layer.DbSchema + "." + layer.DbTable,
								Filter:      []interface{}{"has", "point_count"},
								Paint: models.CircleLayerPaint{
									CircleColor: []interface{}{
										"step",
										[]interface{}{"get", "point_count"},
										"#05a41b",
										100,
										"#02663a",
										750,
										"#024f34",
									},
									CircleRadius: []interface{}{
										"step",
										[]interface{}{"get", "point_count"},
										20,
										100,
										30,
										750,
										40,
									},
									CircleOpacity:       1,
									CircleStrokeWidth:   5,
									CircleStrokeColor:   "#05a41b",
									CircleStrokeOpacity: 0.4,
								},
							}
							style.Layers = append(style.Layers, clusterCircleLayer)

							clusterCountLayer := models.SymbolLayer{
								ID:          layer.ID + "-cluster-count",
								Type:        "symbol",
								Source:      layer.ID,
								SourceLayer: layer.DbSchema + "." + layer.DbTable,
								Filter:      []interface{}{"has", "point_count"},
								Layout: models.SymbolLayerLayout{
									TextField:  []interface{}{"get", "point_count_abbreviated"},
									TextFont:   []string{"Noto Sans Bold"},
									TextSize:   12,
									TextOffset: []float64{0, 0},
									TextAnchor: "center",
								},
								Paint: models.SymbolLayerPaint{
									TextColor: "#ffffff",
								},
							}
							style.Layers = append(style.Layers, clusterCountLayer)
						}

						if generate {
							outputDir := fmt.Sprintf("./public/map/%s/sprite/images", category.MapID)
							err := os.MkdirAll(outputDir, os.ModePerm)
							if err != nil {
								return style, errors.New("Error creating output directory")
							}

							markerPath := *layer.Legends[0].Marker
							outputFile := fmt.Sprintf("%s/%s.png", outputDir, layer.ID)

							if strings.HasSuffix(markerPath, ".svg") {
								err = sprite.SVGToPNG("./public"+markerPath, outputFile)
								if err != nil {
									return style, fmt.Errorf("error converting SVG to PNG for layer %s (%s), file: %s: %w", layer.LayerTitle, layer.ID, markerPath, err)
								}
							} else if strings.HasSuffix(markerPath, ".png") {
								input, err := os.Open("./public" + markerPath)
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
				}
			case "LineString":
				// Define line layer style using line color, width, and other properties
				if len(layer.Legends) >= 1 {
					var lineColor *string
					if layer.Legends[0].StrokeColor != nil {
						lineColor = layer.Legends[0].StrokeColor
					} else if layer.Legends[0].FillColor != nil {
						lineColor = layer.Legends[0].FillColor
					}

					if lineColor != nil {
						lineLayer := models.LineLayer{
							ID:          layer.ID,
							Type:        "line",
							Source:      layer.ID, // Use category source
							SourceLayer: layer.DbSchema + "." + layer.DbTable,
							Paint: models.LineLayerPaint{
								LineColor: *lineColor,
								LineWidth: 2.0,
							},
						}
						style.Layers = append(style.Layers, lineLayer)
					}

				}

			case "Polygon":
				if len(layer.Legends) >= 1 {
					// Add Fill Layer if FillColor exists
					if layer.Legends[0].FillColor != nil {
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
					}

					// Add Line Layer (Stroke) if StrokeColor exists
					if layer.Legends[0].StrokeColor != nil {
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
