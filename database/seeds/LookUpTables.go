package seeds

import (
	"github.com/lambda-platform/lambda/DB"
	"log"
)

func SeedLookupTables() {
	// Seed lut_line_style table
	lineStyles := []struct {
		LineStyle      string
		LineStyleTitle string
	}{
		{"dotted", "Цэгтэй зураас"},
		{"dashed", "Тасархай зураас"},
		{"double", "Давхар зураас"},
		{"solid", "Зураас"},
	}

	for _, style := range lineStyles {
		query := `
		INSERT INTO "map_server"."lut_line_style" ("line_style", "line_style_title")
		VALUES (?, ?)
		ON CONFLICT ("line_style") DO NOTHING;
		`
		if err := DB.DB.Exec(query, style.LineStyle, style.LineStyleTitle).Error; err != nil {
			log.Printf("Failed to seed lut_line_style: %v", err)
		}
	}

	// Seed lut_polygon_style table
	polygonStyles := []struct {
		PolygonStyle      string
		PolygonStyleTitle string
	}{
		{"fill", "Өнгөөр будах"},
		{"cross_lines", "Хөндлөн зураас"},
		{"vertical_lines", "Босоо зураас"},
		{"rigth_lines", "Баруун тийш налсан зураас"},
		{"left_lines", "Зүүн тийш налсан зураас"},
		{"dotted", "Цэгтэй"},
	}

	for _, style := range polygonStyles {
		query := `
		INSERT INTO "map_server"."lut_polygon_style" ("polygon_style", "polygon_style_title")
		VALUES (?, ?)
		ON CONFLICT ("polygon_style") DO NOTHING;
		`
		if err := DB.DB.Exec(query, style.PolygonStyle, style.PolygonStyleTitle).Error; err != nil {
			log.Printf("Failed to seed lut_polygon_style: %v", err)
		}
	}
}
