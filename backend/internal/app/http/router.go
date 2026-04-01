package http

import (
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"

	appconfig "github.com/wecrazy/smart-inventory-core-system/backend/internal/app/config"
	"github.com/wecrazy/smart-inventory-core-system/backend/internal/app/service"
)

func NewApp(_ appconfig.Config, service *service.Service) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName:       "smart-inventory-core-system",
		CaseSensitive: false,
	})

	app.Use(cors.New())
	registerSwagger(app)

	handler := newHandler(service)

	api := app.Group("/api/v1")
	api.Get("/health", handler.healthCheck)

	api.Get("/inventory", handler.listInventory)
	api.Post("/inventory", handler.createInventory)
	api.Post("/inventory/adjustments", handler.adjustInventory)

	api.Post("/stock-in", handler.createStockIn)
	api.Get("/stock-in", handler.listStockIn)
	api.Get("/stock-in/:id", handler.getStockIn)
	api.Patch("/stock-in/:id/status", handler.updateStockInStatus)
	api.Post("/stock-in/:id/cancel", handler.cancelStockIn)

	api.Post("/stock-out", handler.createStockOut)
	api.Get("/stock-out", handler.listStockOut)
	api.Get("/stock-out/:id", handler.getStockOut)
	api.Patch("/stock-out/:id/status", handler.updateStockOutStatus)
	api.Post("/stock-out/:id/cancel", handler.cancelStockOut)

	api.Get("/reports/export", handler.exportReports)
	api.Get("/reports", handler.listReports)

	return app
}
