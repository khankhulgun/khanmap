package migrations

import (
	"github.com/khankhulgun/khanmap/models"
	"github.com/lambda-platform/lambda/DB"
	"log"
)

func Migrate() {
	// Create the schema if it doesn't exist
	createSchema := `
	CREATE SCHEMA IF NOT EXISTS map_server;
	`

	err := DB.DB.Exec(createSchema).Error
	if err != nil {
		log.Fatalf("Failed to create schema: %v", err)
	}
	DB.DB.AutoMigrate(
		&models.SubMapLayerCategories{},
		&models.MapFilters{},
		&models.MapLayersForTile{},
		&models.MapLayerCategory{},
		&models.MapLayers{},
		&models.MapLayerLegends{},
		&models.SubMapLayerPermissions{},
		&models.SubMapLayerFilters{},
		&models.SubMapLayerAdminFilters{},
	)
	// Create the view
	createView := `
	CREATE OR REPLACE VIEW map_server.view_map_layer_categories AS
	SELECT 
		sub_map_layer_categories.map_id,
		sub_map_layer_categories.category_order,
		map_layer_category.id,
		map_layer_category.icon,
		map_layer_category.is_active,
		map_layer_category.is_visible,
		map_layer_category.layer_category
	FROM 
		map_server.sub_map_layer_categories
	LEFT JOIN 
		map_server.map_layer_category 
	ON 
		sub_map_layer_categories.map_category_id = map_layer_category.id;

	CREATE OR REPLACE VIEW map_server.view_map_filters AS
	SELECT map_filters.id,
		map_filters.map_id,
		map_filters.label,
		map_filters.value_field,
		map_filters.label_field,
		map_filters."table",
		map_filters.parent_filter_in_table,
		map_filters.parent_filter_id,
		map_filters.filter_order,
		map_filters.created_at,
		map_filters.updated_at,
		map_filters.deleted_at,
		map.map
	FROM map_server.map_filters
		LEFT JOIN map_server.map ON map_filters.map_id = map.id;

	CREATE OR REPLACE VIEW map_server.view_map_layer_category AS
	SELECT id,
		icon,
		is_active,
		is_visible,
		layer_category
	   FROM map_server.map_layer_category;

	CREATE OR REPLACE VIEW map_server.view_map_layers AS
	 SELECT map_layers.id,
		map_layers.db_table,
		map_layers.geometry_type,
		map_layers.geometry_fieldname,
		map_layers.id_fieldname,
		map_layers.db_schema,
		map_layers.column_selects,
		map_layers.is_active,
		map_layers.is_public,
		map_layers.is_visible,
		map_layers.layer_order,
		map_layers.map_layer_category_id,
		map_layers.layer_title,
		map_layers.description,
		map_layers.popup_template,
		map_layers.unique_value_field,
		map_layers.is_overlap,
		map_layers.is_permission,
		map_layers.soum_id_field,
		map_layers.bagh_id_field,
		map_layer_category.layer_category
	   FROM map_server.map_layers
		 LEFT JOIN map_server.map_layer_category ON map_layers.map_layer_category_id = map_layer_category.id;
	`
	DB.DB.AutoMigrate(
		&models.Map{},
	)
	// Execute the SQL for the view
	err = DB.DB.Exec(createView).Error
	if err != nil {
		log.Fatalf("Failed to create view: %v", err)
	}

	MigrateLookupTables()

}
