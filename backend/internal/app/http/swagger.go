package http

import (
	swaggo "github.com/gofiber/contrib/v3/swaggo"

	"github.com/gofiber/fiber/v3"

	apidocs "github.com/wecrazy/smart-inventory-core-system/backend/docs/swagger"
)

func registerSwagger(app *fiber.App) {
	apidocs.SwaggerInfo.BasePath = "/api/v1"
	apidocs.SwaggerInfo.Host = ""
	apidocs.SwaggerInfo.Schemes = []string{"http", "https"}

	app.Get("/swagger/*", swaggo.New(swaggo.Config{
		Title:        "Smart Inventory Core System API Docs",
		URL:          "/swagger/doc.json",
		DeepLinking:  true,
		DocExpansion: "list",
	}))
}
