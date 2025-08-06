// Package config provides configuration settings for the sefud application.
// It handles loading environment variables and provides structured access to configuration values.
package config

import (
	"os"
	"strconv"

	"github.com/charmbracelet/log"
	"github.com/joho/godotenv"
)

type Config struct {
	Database DBConfig
	App      AppConfig
	R2       R2Config
}

func NewConfig() *Config {
	err := godotenv.Load()
	if err != nil {
		log.Warn("Error loading .env file, falling back to default environment variables")
	}

	return &Config{
		Database: LoadDBConfig(),
		App:      LoadAppConfig(),
		R2:       LoadR2Config(),
	}
}

func GetEnv(key, defaultValue string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		return defaultValue
	}
	return value
}

func GetEnvAsInt64(name string, defaultValue int64) int64 {
	valueStr := GetEnv(name, "")
	if value, err := strconv.ParseInt(valueStr, 10, 64); err == nil {
		return value
	}
	return defaultValue
}
