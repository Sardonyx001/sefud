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
}

func NewConfig() *Config {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file, falling back to default environment variables")
	}

	return &Config{
		Database: LoadDBConfig(),
		App:      LoadAppConfig(),
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
