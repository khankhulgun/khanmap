package khanmap

import (
	"github.com/gofiber/fiber/v2"
	"github.com/khankhulgun/khanmap/controllers"
	"github.com/khankhulgun/khanmap/tiles"
	"github.com/lambda-platform/lambda/agent/agentMW"
)

func Set(app *fiber.App) {
	app.Get("/tiles/:layer/:z/:x/:y.pbf", tiles.VectorTileHandler)
	app.Get("/tiles/:layer/:z/:x/:y/:token.pbf", tiles.VectorTileHandlerWithToken)
	app.Get("/saved-tiles/:layer/:z/:x/:y.pbf", tiles.SaveVectorTileHandler)
	app.Get("/save-tile/:layer", tiles.SaveHandler)

	a := app.Group("/mapserver/api")
	a.Get("/geometry-tables", agentMW.IsLoggedIn(), controllers.GeometryTables)
	a.Get("/table-columns/:schema/:table", agentMW.IsLoggedIn(), controllers.TableColumns)
	a.Get("/map/:id", controllers.GetMapLayers)

	a.Post("/spatial/:layer/:relationship", controllers.Spatial)

}
