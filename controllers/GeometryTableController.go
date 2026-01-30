package controllers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/khankhulgun/khanmap/models"
	"github.com/lambda-platform/lambda/DB"
)

func GeometryTables(c *fiber.Ctx) error {

	var geometryTables []models.GeometryTable
	if err := DB.DB.Raw(`SELECT f_table_schema AS schema, f_table_name, f_geometry_column, coord_dimension, srid, type FROM geometry_columns`).Scan(&geometryTables).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(geometryTables)

}

func TableColumns(c *fiber.Ctx) error {

	schema := c.Params("schema")
	tableName := c.Params("table")
	var columns []models.TableColumn
	if err := DB.DB.Raw(`SELECT 
    a.attname AS column_name,
    format_type(a.atttypid, a.atttypmod) AS data_type
FROM 
    pg_attribute a
JOIN 
    pg_class c ON a.attrelid = c.oid
JOIN 
    pg_namespace n ON c.relnamespace = n.oid
WHERE 
    c.relname = ?      
    AND n.nspname = ?   
    AND a.attnum > 0 
    AND NOT a.attisdropped;`, tableName, schema).Scan(&columns).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(columns)

}
