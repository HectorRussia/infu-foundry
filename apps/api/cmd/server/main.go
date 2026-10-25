package main

import (
	"context"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v3"
	"infu-foundry/apps/api/internal/config"
	"infu-foundry/apps/api/internal/database"
	"infu-foundry/apps/api/internal/httpapi"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if cfg.DatabaseURL != "" {
		db, err := database.Open(ctx, cfg.DatabaseURL)
		if err != nil {
			return err
		}
		defer db.Close()
		logger.Info("database connected")
		// Inject db.ORM into repositories when adding the real content/auth routes.
	} else {
		logger.Info("database not configured; running foundation API")
	}
	app := httpapi.New(logger, cfg.Environment == "development")
	return app.Listen(":"+cfg.Port, fiber.ListenConfig{
		GracefulContext:       ctx,
		ShutdownTimeout:       10 * time.Second,
		DisableStartupMessage: true,
		ListenerNetwork:       "tcp",
		ListenerAddrFunc: func(addr net.Addr) {
			logger.Info("api listening", "address", addr.String(), "framework", "fiber", "environment", cfg.Environment)
		},
	})
}
