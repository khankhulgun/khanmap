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
	if err := DB.DB.Raw(`SELECT column_name, data_type FROM information_schema.columns WHERE table_schema = ? AND table_name = ?`, schema, tableName).Scan(&columns).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(columns)

}
