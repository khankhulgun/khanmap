package khanmap

import (
	"github.com/gofiber/fiber/v2"
	"github.com/khankhulgun/khanmap/controllers"
	"github.com/khankhulgun/khanmap/tiles"
	"github.com/lambda-platform/lambda/agent/agentMW"
)

func Set(app *fiber.App) {
	app.Get("/tiles/:layer/:z/:x/:y.pbf", tiles.VectorTileHandler)

	a := app.Group("/mapserver/api")
	a.Get("/geometry-tables", agentMW.IsLoggedIn(), controllers.GeometryTables)
	a.Get("/table-columns/:schema/:table", agentMW.IsLoggedIn(), controllers.TableColumns)
	a.Get("/map/:id", controllers.GetMapLayers)

}
