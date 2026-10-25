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
	"infu-foundry/apps/api/internal/auth"
	"infu-foundry/apps/api/internal/config"
	"infu-foundry/apps/api/internal/contents"
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
	opts := httpapi.Options{Development: cfg.Environment == "development", GatewaySecret: cfg.GatewaySecret}
	if cfg.DatabaseURL != "" {
		db, err := database.Open(ctx, cfg.DatabaseURL)
		if err != nil {
			return err
		}
		defer db.Close()
		logger.Info("database connected")
		opts.Contents = contents.NewStore(db.ORM)
		opts.Auth = auth.NewStore(db.ORM)
	} else {
		logger.Info("database not configured; running foundation API")
	}
	if !opts.Development && opts.GatewaySecret == "" {
		logger.Warn("GATEWAY_SECRET not set; the API accepts requests from any origin host")
	}
	app := httpapi.New(logger, opts)
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
