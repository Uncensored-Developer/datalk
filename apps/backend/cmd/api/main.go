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
	"github.com/Uncensored-Developer/datalk/apps/backend/pkg/distlock/postgres"
	"github.com/Uncensored-Developer/datalk/apps/backend/pkg/logger"
	"github.com/Uncensored-Developer/datalk/apps/backend/pkg/pubsub"
	pubsubpostgres "github.com/Uncensored-Developer/datalk/apps/backend/pkg/pubsub/postgres"
	echoauth "github.com/Uncensored-Developer/datalk/apps/backend/servers/echo/api/authenticator"
	authhandlers "github.com/Uncensored-Developer/datalk/apps/backend/servers/echo/handlers/resources/auth"
	connectionhandlers "github.com/Uncensored-Developer/datalk/apps/backend/servers/echo/handlers/resources/connections"
	userhandlers "github.com/Uncensored-Developer/datalk/apps/backend/servers/echo/handlers/users"
	connectionsservice "github.com/Uncensored-Developer/datalk/apps/backend/services/connections"
	schemasservice "github.com/Uncensored-Developer/datalk/apps/backend/services/schemas"
	usersservice "github.com/Uncensored-Developer/datalk/apps/backend/services/users"
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
	if cfg.JWTSecret == "" {
		logger.Fatal("JWT_SECRET is required")
	}

	log := logger.SetupLogger(cfg)
	slog.SetDefault(log)

	ctx, conn, err := setupDB(ctx, cfg, log)
	if err != nil {
		logger.Fatal("failed to connect to DB", logger.Err(err))
	}

	e := echo.New()
	e.Use(requestLoggingMiddleware(log))
	e.Use(middleware.Recover())

	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{"*"},
	}))

	locker := postgres.NewDistributedLocker(conn)
	pubsubBus := pubsubpostgres.NewBus(conn, pubsubConnInfo(cfg))
	defer pubsubBus.Close()
	pubsub.RegisterPublisher(pubsubBus)

	usersService := usersservice.New(cfg, conn)
	connectionsService := connectionsservice.New(cfg, conn)
	schemasService, err := schemasservice.New(ctx, conn, cfg, log, locker, connectionsService.API)
	if err != nil {
		logger.Fatal("failed to create schemas service", logger.Err(err))
	}

	jwtAuthenticator := echoauth.NewJWTAuthenticator(cfg, usersService.API)
	authMiddleware := echoauth.Middleware(jwtAuthenticator)
	authhandlers.New(usersService.API, log).RegisterPublic(e.Group("/api"))

	protectedAPI := e.Group("/api", authMiddleware)
	authhandlers.New(usersService.API, log).RegisterProtected(protectedAPI)
	userhandlers.New(usersService.API, log).Register(protectedAPI.Group("/users"))
	connectionhandlers.New(connectionsService.API, log).Register(protectedAPI)

	embeddingSubscription, err := schemasService.SubscribeSnapshotEmbedding(ctx, pubsubBus)
	if err != nil {
		logger.Fatal("failed to subscribe snapshot embedding handler", logger.Err(err))
	}
	if embeddingSubscription != nil {
		defer embeddingSubscription.Close(context.Background())
	}

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

func pubsubConnInfo(cfg config.Config) string {
	return fmt.Sprintf(
		"user=%s password=%s host=%s port=%d dbname=%s sslmode=%s search_path=%s",
		cfg.DBUser,
		cfg.DBPassword,
		cfg.DBHost,
		cfg.DBPort,
		cfg.DBName,
		cfg.DBSSLMode,
		cfg.DBSchema,
	)
}
