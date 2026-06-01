package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"nix-backend/internal/config"
	"nix-backend/internal/database"
	"nix-backend/internal/handlers"
	"nix-backend/internal/middleware"
	"nix-backend/internal/repository"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()
	log.Printf("starting nix-backend (port=%s gin_mode=%s)", cfg.Port, cfg.GinMode)

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
	authHandler := handlers.NewAuthHandler(cfg, userRepo)
	txHandler := handlers.NewTransactionHandler(txRepo)

	gin.SetMode(cfg.GinMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.RequestLogger())
	r.Use(corsMiddleware())

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	api := r.Group("/api")
	{
		api.POST("/auth/register", authHandler.Register)
		api.POST("/auth/login", authHandler.Login)

		protected := api.Group("")
		protected.Use(middleware.AuthRequired(cfg))
		{
			protected.GET("/auth/me", authHandler.Me)
			protected.GET("/transactions", txHandler.List)
			protected.GET("/transactions/:id", txHandler.Get)
			protected.POST("/transactions", txHandler.Create)
			protected.PUT("/transactions/:id", txHandler.Update)
			protected.DELETE("/transactions/:id", txHandler.Delete)
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

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	_ = srv.Shutdown(shutdownCtx)
}

func corsMiddleware() gin.HandlerFunc {
	allowedOrigins := map[string]bool{
		"http://localhost:5173":       true,
		"https://nix-one.netlify.app": true,
	}

	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		if allowedOrigins[origin] {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Credentials", "true")
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
