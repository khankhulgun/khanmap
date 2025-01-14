package migrations

import (
	"github.com/lambda-platform/lambda/DB"
	"log"
)

func MigrateLookupTables() {
	// Create lut_line_style table
	createLineStyleTable := `
	CREATE TABLE IF NOT EXISTS "map_server"."lut_line_style" (
		"line_style" VARCHAR(255) COLLATE "pg_catalog"."default" NOT NULL,
		"line_style_title" VARCHAR(255) COLLATE "pg_catalog"."default" NOT NULL,
		CONSTRAINT "lut_line_style_pkey" PRIMARY KEY ("line_style")
	);
	`
	if err := DB.DB.Exec(createLineStyleTable).Error; err != nil {
		log.Fatalf("Failed to create lut_line_style table: %v", err)
	}

	// Create lut_polygon_style table
	createPolygonStyleTable := `
	CREATE TABLE IF NOT EXISTS "map_server"."lut_polygon_style" (
		"polygon_style" VARCHAR(255) COLLATE "pg_catalog"."default" NOT NULL,
		"polygon_style_title" VARCHAR(255) COLLATE "pg_catalog"."default" NOT NULL,
		CONSTRAINT "lut_polygon_style_pkey" PRIMARY KEY ("polygon_style")
	);
	`
	if err := DB.DB.Exec(createPolygonStyleTable).Error; err != nil {
		log.Fatalf("Failed to create lut_polygon_style table: %v", err)
	}
}
