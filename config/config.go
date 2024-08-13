package config

import (
	"os"
	"strconv"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/joho/godotenv"
)

type Config struct {
	AppPort       string
	StoragePath   string
	MimeBlacklist []string
	MaxUploadSize int64
	DatabaseURL   string
}

var AppConfig Config

func LoadConfig() {
	err := godotenv.Load()
	if err != nil {
		log.Warn("Error loading .env file, falling back to default environment variables")
	}

	AppConfig = Config{
		AppPort:       getEnv("SEFUD_APP_PORT", "7000"),
		StoragePath:   getEnv("SEFUD_STORAGE_PATH", "./uploads"),
		MimeBlacklist: strings.Split(getEnv("SEFUD_MIME_BLACKLIST", "application/x-sh"), ","),
		MaxUploadSize: getEnvAsInt64("MAX_CONTENT_LENGTH", 10*1024*1024), // 10MB default
		DatabaseURL:   getEnv("DATABASE_URL", ""),
	}

	log.Info("Configuration loaded successfully")
}

func getEnv(key, defaultValue string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		return defaultValue
	}
	return value
}

func getEnvAsInt64(name string, defaultValue int64) int64 {
	valueStr := getEnv(name, "")
	if value, err := strconv.ParseInt(valueStr, 10, 64); err == nil {
		return value
	}
	return defaultValue
}
