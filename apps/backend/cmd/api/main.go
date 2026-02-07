package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/Uncensored-Developer/datalk/apps/backend/config"
	"github.com/Uncensored-Developer/datalk/apps/backend/db/common"
	"github.com/Uncensored-Developer/datalk/apps/backend/pkg/logger"
	"github.com/alecthomas/kingpin/v2"
	_ "github.com/joho/godotenv/autoload"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

var tryMigrate = kingpin.Flag("try-migrate", "Try to migrate on start").Default("false").Bool()

func main() {
	ctx := context.Background()
	kingpin.Parse()
	cfg := config.MustLoad()

	log := logger.SetupLogger(cfg)
	slog.SetDefault(log)

	ctx, _, err := setupDB(ctx, cfg, log)
	if err != nil {
		logger.Fatal("failed to connect to DB", logger.Err(err))
	}

	e := echo.New()
	e.Use(middleware.Recover())

	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{"*"},
	}))

	ctx, stop := signal.NotifyContext(ctx, os.Interrupt)
	defer stop()

	// Start server
	go func() {
		apiPort := fmt.Sprintf(":%d", cfg.ApiPort)
		if err := e.Start(apiPort); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal("failed to start server", logger.Err(err))
		}
	}()

	// Wait for the interrupt signal to gracefully shut down the server with a timeout of 10 seconds.
	<-ctx.Done()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		logger.Fatal("failed to shutdown server", logger.Err(err))
	}
}

func setupDB(ctx context.Context, cfg config.Config, log *slog.Logger) (context.Context, *sql.DB, error) {
	conn, err := common.DBFromConfig(cfg, cfg.DBSchema, *tryMigrate, log)
	if err != nil {
		logger.Fatal("failed to connect to DB", logger.Err(err))
	}
	return ctx, conn, nil
}
