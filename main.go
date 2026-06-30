package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"nix-backend/internal/config"
	"nix-backend/internal/database"
	"nix-backend/internal/handlers"
	"nix-backend/internal/keepalive"
	"nix-backend/internal/middleware"
	"nix-backend/internal/repository"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()
	log.Printf("starting v0.0.1 nix-backend (port=%s gin_mode=%s)", cfg.Port, cfg.GinMode)

	if cfg.MongoURI == "" || cfg.MongoURI == "mongodb://localhost:27017/nix" {
		if os.Getenv("MONGODB_URI") == "" {
			log.Fatal("mongodb: MONGODB_URI is not set — add it in Render Environment Variables")
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	client, err := database.Connect(ctx, cfg.MongoURI)
	cancel()
	if err != nil {
		log.Fatalf("mongodb: connect failed: %v (check MONGODB_URI and Atlas Network Access 0.0.0.0/0)", err)
	}
	log.Print("mongodb: connected")
	defer func() {
		disconnectCtx, disconnectCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer disconnectCancel()
		_ = client.Disconnect(disconnectCtx)
	}()

	dbName := "nix"
	if idx := strings.LastIndex(cfg.MongoURI, "/"); idx != -1 {
		rest := cfg.MongoURI[idx+1:]
		if name := strings.SplitN(rest, "?", 2)[0]; name != "" {
			dbName = name
		}
	}
	db := database.DB(client, dbName)

	userRepo := repository.NewUserRepository(db)
	txRepo := repository.NewTransactionRepository(db)
	titleRepo := repository.NewRecordTitleRepository(db)

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	authHandler := handlers.NewAuthHandler(cfg, userRepo, logger)
	txHandler := handlers.NewTransactionHandler(txRepo)
	titleHandler := handlers.NewRecordTitleHandler(titleRepo)

	indexCtx, indexCancel := context.WithTimeout(context.Background(), 10*time.Second)
	if err := titleRepo.EnsureIndexes(indexCtx); err != nil {
		log.Printf("mongodb: record_titles index: %v", err)
	}
	indexCancel()

	gin.SetMode(cfg.GinMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())
	r.Use(corsMiddleware(cfg.CORSOrigin))

	api := r.Group("/api")
	{
		api.GET("/health", func(c *gin.Context) {
			handlers.OK(c, gin.H{"status": "ok"})
		})

		auth := api.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
			auth.POST("/reset-password", authHandler.ResetPassword)
		}

		protected := api.Group("")
		protected.Use(middleware.AuthRequired(cfg))
		{
			protectedAuth := protected.Group("/auth")
			{
				protectedAuth.GET("/me", authHandler.Me)
			}

			transactions := protected.Group("/transactions")
			{
				transactions.GET("", txHandler.List)
				transactions.GET("/:id", txHandler.Get)
				transactions.POST("", txHandler.Create)
				transactions.PUT("/:id", txHandler.Update)
				transactions.DELETE("/:id", txHandler.Delete)
			}

			titles := protected.Group("/titles")
			{
				titles.GET("", titleHandler.List)
				titles.POST("", titleHandler.Create)
				titles.PUT("/:id", titleHandler.Update)
				titles.DELETE("/:id", titleHandler.Delete)
			}
		}
	}

	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: r,
	}

	go func() {
		log.Printf("nix-backend listening on :%s (db=%s)", cfg.Port, dbName)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	keepAliveCtx, keepAliveCancel := context.WithCancel(context.Background())
	defer keepAliveCancel()
	keepalive.Start(keepAliveCtx, cfg.PublicURL)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	keepAliveCancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	_ = srv.Shutdown(shutdownCtx)
}

func corsMiddleware(corsOrigin string) gin.HandlerFunc {
	allowedOrigins := buildAllowedOrigins(corsOrigin)
	log.Printf("cors: allowed origins %v", originKeys(allowedOrigins))

	return func(c *gin.Context) {
		origin := normalizeOrigin(c.Request.Header.Get("Origin"))

		if allowedOrigins[origin] {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Vary", "Origin")
		}

		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization")
		c.Header("Access-Control-Expose-Headers", "Content-Length")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

func buildAllowedOrigins(corsOrigin string) map[string]bool {
	allowed := map[string]bool{
		"http://localhost:5173":       true,
		"https://nix-one.netlify.app": true,
	}
	for _, o := range strings.Split(corsOrigin, ",") {
		if o = normalizeOrigin(o); o != "" {
			allowed[o] = true
		}
	}
	return allowed
}

func normalizeOrigin(origin string) string {
	return strings.TrimRight(strings.TrimSpace(origin), "/")
}

func originKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
