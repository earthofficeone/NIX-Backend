package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port       string
	MongoURI   string
	JWTSecret  string
	CORSOrigin string
	GinMode    string
}

func Load() Config {
	_ = godotenv.Load()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mongoURI := os.Getenv("MONGODB_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://localhost:27017/nix"
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "nix-dev-secret-change-in-production"
	}

	corsOrigin := os.Getenv("CORS_ORIGIN")
	if corsOrigin == "" {
		corsOrigin = "http://localhost:5173"
	}

	ginMode := os.Getenv("GIN_MODE")
	if ginMode == "" {
		ginMode = "debug"
	}

	return Config{
		Port:       port,
		MongoURI:   mongoURI,
		JWTSecret:  jwtSecret,
		CORSOrigin: corsOrigin,
		GinMode:    ginMode,
	}
}
