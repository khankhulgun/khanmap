package khanmap

import (
	"github.com/gofiber/fiber/v2"
	"github.com/khankhulgun/khanmap/controllers"
	"github.com/khankhulgun/khanmap/database/migrations"
	"github.com/khankhulgun/khanmap/database/seeds"
	"github.com/khankhulgun/khanmap/tiles"
	"github.com/lambda-platform/lambda/agent/agentMW"
	"github.com/lambda-platform/lambda/config"
)

func Set(app *fiber.App) {
	app.Get("/tiles/:layer/:z/:x/:y.pbf", tiles.VectorTileHandler)
	app.Get("/tiles-with-permission/:layer/:z/:x/:y.pbf", agentMW.IsLoggedIn(), tiles.VectorTileHandlerWithPermission)
	app.Get("/saved-tiles/:layer/:z/:x/:y.pbf", tiles.SaveVectorTileHandler)
	app.Get("/save-tile/:layer", tiles.SaveHandler)

	a := app.Group("/mapserver/api")
	a.Get("/geometry-tables", agentMW.IsLoggedIn(), controllers.GeometryTables)
	a.Get("/table-columns/:schema/:table", agentMW.IsLoggedIn(), controllers.TableColumns)
	a.Get("/map/:id", controllers.GetMapLayers)
	a.Post("/spatial/:layer/:relationship", controllers.Spatial)
	a.Post("/map-data", controllers.GetMapData)
	a.Get("/filter-options", controllers.FilterOptions)

	if config.Config.App.Migrate == "true" {
		migrations.Migrate()
	}
	if config.Config.App.Seed == "true" {
		seeds.Seed()
	}
}
