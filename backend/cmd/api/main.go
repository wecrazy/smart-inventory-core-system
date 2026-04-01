package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	appconfig "github.com/wecrazy/smart-inventory-core-system/backend/internal/app/config"
	transport "github.com/wecrazy/smart-inventory-core-system/backend/internal/app/http"
	"github.com/wecrazy/smart-inventory-core-system/backend/internal/app/service"
	"github.com/wecrazy/smart-inventory-core-system/backend/internal/platform/postgres"
)

// @title           Smart Inventory Core System API
// @version         1.0
// @description     Inventory, stock-in, stock-out, adjustment, and reporting API for the Smart Inventory Core System assessment.
// @description     Swagger docs are generated from Fiber handler annotations and served by the backend itself.
// @BasePath        /api/v1
// @schemes         http
// @accept          json
// @produce         json
// @tag.name        System
// @tag.description Operational endpoints such as health checks and documentation.
// @tag.name        Inventory
// @tag.description Inventory master data and auditable stock adjustments.
// @tag.name        Stock In
// @tag.description Inbound transaction lifecycle from creation to completion or cancellation.
// @tag.name        Stock Out
// @tag.description Reservation-safe outbound transaction lifecycle.
// @tag.name        Reports
// @tag.description Done-only reporting endpoints.

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg := appconfig.Load()

	pool, err := postgres.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}
	defer pool.Close()

	repository := postgres.NewRepository(pool)
	inventoryService := service.New(repository, repository)

	app := transport.NewApp(cfg, inventoryService)

	go func() {
		if err := app.Listen(cfg.Address()); err != nil {
			log.Printf("fiber server stopped: %v", err)
			stop()
		}
	}()

	log.Printf("smart inventory api listening on %s", cfg.Address())

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := app.ShutdownWithContext(shutdownCtx); err != nil {
		log.Printf("shutdown error: %v", err)
	}
}
