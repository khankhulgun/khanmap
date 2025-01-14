package seeds

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/lambda-platform/lambda/DB"
	krudModels "github.com/lambda-platform/lambda/krud/models"
	puzzleModels "github.com/lambda-platform/lambda/models"
)

// Seed initializes the database with data from vb_schemas.json
func Seed() {
	absolutePath := AbsolutePath()

	// Define the file path
	fileName := "vb_schemas.json"
	filePath := filepath.Join(absolutePath, fileName)

	// Load the data from the JSON file
	vbSchemas, err := LoadVBSchemas(filePath)
	if err != nil {
		log.Fatalf("Failed to load seed data: %v", err)
	}

	// Seed the data into the database
	err = SeedVBSchemas(vbSchemas)
	if err != nil {
		log.Fatalf("Failed to seed data: %v", err)
	}

	fmt.Println("Seed data successfully loaded and updated into the database.")
}

// LoadVBSchemas loads the vb_schemas.json file and unmarshals it into a slice of VBSchema
func LoadVBSchemas(filePath string) ([]puzzleModels.VBSchema, error) {
	var vbSchemas []puzzleModels.VBSchema

	// Open the JSON file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("error opening file %s: %w", filePath, err)
	}
	defer file.Close()

	// Parse the JSON data
	jsonParser := json.NewDecoder(file)
	err = jsonParser.Decode(&vbSchemas)
	if err != nil {
		return nil, fmt.Errorf("error decoding JSON data: %w", err)
	}

	return vbSchemas, nil
}

// SeedVBSchemas inserts or updates vb_schemas and creates/updates related Krud records
func SeedVBSchemas(vbSchemas []puzzleModels.VBSchema) error {
	legendFormID := 0
	for _, vb := range vbSchemas {

		// Update schema for "Давхарга" with type "form"
		if vb.Name == "Давхарга" && vb.Type == "form" {
			vb.Schema = strings.Replace(vb.Schema, `"formId":54`, fmt.Sprintf(`"formId":%d`, legendFormID), 1)

		}

		// Find existing vb_schema by name
		var existingVB puzzleModels.VBSchema
		err := DB.DB.Where("name = ? AND type = ?", vb.Name, vb.Type).First(&existingVB).Error
		if err != nil {
			// If not found, insert the vb_schema
			if err := DB.DB.Create(&vb).Error; err != nil {
				return fmt.Errorf("error creating vb_schema with name %s: %w", vb.Name, err)
			}
			existingVB = vb
		}

		// Find or create Krud entry
		var krud krudModels.Krud
		err = DB.DB.Where("title = ?", vb.Name).First(&krud).Error
		if err != nil {
			// If Krud doesn't exist, create it
			krud = krudModels.Krud{
				Title:     vb.Name,
				Template:  "canvas",
				Grid:      0,
				Form:      0,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
		}

		// Update Krud's Grid or Form based on vb.Type
		if vb.Type == "grid" {
			krud.Grid = int(existingVB.ID)
		} else if vb.Type == "form" {
			krud.Form = int(existingVB.ID)
		}
		// Skip entries with the name "Таних тэмдэг"
		if vb.Name == "Таних тэмдэг" && vb.Type == "form" {
			legendFormID = int(vb.ID)
			continue
		}
		// Save the updated or newly created Krud entry
		if err := DB.DB.Save(&krud).Error; err != nil {
			return fmt.Errorf("error saving Krud for vb_schema with name %s: %w", vb.Name, err)
		}
	}
	return nil
}

// AbsolutePath returns the absolute path to the current file's directory
func AbsolutePath() string {
	_, fileName, _, _ := runtime.Caller(0)
	return filepath.Dir(fileName) + "/"
}
