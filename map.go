package khanmap

import (
	"github.com/gofiber/fiber/v2"
	"github.com/khankhulgun/khanmap/tiles"
)

func Set(app *fiber.App) {
	app.Get("/tiles/:layer/:z/:x/:y.pbf", tiles.VectorTileHandler)
}
