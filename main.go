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
	"nix-backend/internal/email"
	"nix-backend/internal/handlers"
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
	resetRepo := repository.NewPasswordResetRepository(db)
	txRepo := repository.NewTransactionRepository(db)
	titleRepo := repository.NewRecordTitleRepository(db)

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	var mailer email.Sender
	if cfg.EmailConfigured() {
		cfg.LogEmailConfig(logger)
		mailer = email.NewSMTPSender(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUser, cfg.SMTPPass, cfg.SMTPFrom)
		log.Print("email: SMTP configured")
	} else {
		cfg.LogEmailConfig(logger)
		mailer = email.NewLogSender(logger)
		log.Print("email: SMTP not configured — OTP will be logged to stdout (dev mode)")
	}

	authHandler := handlers.NewAuthHandler(cfg, userRepo, resetRepo, mailer, logger)
	txHandler := handlers.NewTransactionHandler(txRepo)
	titleHandler := handlers.NewRecordTitleHandler(titleRepo)

	indexCtx, indexCancel := context.WithTimeout(context.Background(), 10*time.Second)
	if err := titleRepo.EnsureIndexes(indexCtx); err != nil {
		log.Printf("mongodb: record_titles index: %v", err)
	}
	if err := resetRepo.EnsureIndexes(indexCtx); err != nil {
		log.Printf("mongodb: password_resets index: %v", err)
	}
	indexCancel()

	gin.SetMode(cfg.GinMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.LoggerWithConfig(gin.LoggerConfig{SkipPaths: []string{"/health"}}))
	r.Use(corsMiddleware())

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	api := r.Group("/api")
	{
		auth := api.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
			auth.POST("/forgot-password", authHandler.ForgotPassword)
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
