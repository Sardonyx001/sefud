package config

import (
	"strings"
)

type AppConfig struct {
	Port          string
	StoragePath   string
	MimeBlacklist []string
	MaxUploadSize int64
	DatabaseURL   string
}

func LoadAppConfig() AppConfig {
	return AppConfig{
		Port:          GetEnv("SEFUD_APP_PORT", "7000"),
		StoragePath:   GetEnv("SEFUD_STORAGE_PATH", "./uploads"),
		MimeBlacklist: strings.Split(GetEnv("SEFUD_MIME_BLACKLIST", "application/x-sh"), ","),
		MaxUploadSize: GetEnvAsInt64("SEFUD_MAX_UPLOAD_SIZE", 10*1024*1024), // 10MB default
	}
}
