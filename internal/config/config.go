package config

import (
	"log/slog"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Port            string
	MongoURI        string
	JWTSecret       string
	CORSOrigin      string
	GinMode         string
	SMTPHost        string
	SMTPPort        string
	SMTPUser        string
	SMTPPass        string
	SMTPFrom        string
	ResetMasterCode string
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
	corsOrigin = strings.TrimRight(strings.TrimSpace(corsOrigin), "/")

	ginMode := os.Getenv("GIN_MODE")
	if ginMode == "" {
		ginMode = "debug"
	}

	smtpPort := os.Getenv("SMTP_PORT")
	if smtpPort == "" {
		smtpPort = "587"
	}

	resetMasterCode := os.Getenv("RESET_MASTER_CODE")
	if resetMasterCode == "" {
		resetMasterCode = "230644"
	}

	return Config{
		Port:            port,
		MongoURI:        mongoURI,
		JWTSecret:       jwtSecret,
		CORSOrigin:      corsOrigin,
		GinMode:         ginMode,
		SMTPHost:        os.Getenv("SMTP_HOST"),
		SMTPPort:        smtpPort,
		SMTPUser:        os.Getenv("SMTP_USER"),
		SMTPPass:        os.Getenv("SMTP_PASSWORD"),
		SMTPFrom:        os.Getenv("SMTP_FROM"),
		ResetMasterCode: resetMasterCode,
	}

}

func (c Config) EmailConfigured() bool {
	return c.SMTPHost != "" && c.SMTPFrom != ""
}

func (c Config) LogEmailConfig(logger *slog.Logger) {
	logger.Info("email env",
		"smtp_host", emptyPlaceholder(c.SMTPHost),
		"smtp_port", c.SMTPPort,
		"smtp_user", emptyPlaceholder(c.SMTPUser),
		"smtp_password", maskSecret(c.SMTPPass),
		"smtp_from", emptyPlaceholder(c.SMTPFrom),
		"email_configured", c.EmailConfigured(),
	)
}

func emptyPlaceholder(v string) string {
	if v == "" {
		return "(empty)"
	}
	return v
}

func maskSecret(s string) string {
	if s == "" {
		return "(empty)"
	}
	if len(s) <= 4 {
		return "****"
	}
	return s[:2] + "****" + s[len(s)-2:]
}
