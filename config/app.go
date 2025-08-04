// Package config provides configuration settings for the application.
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

type R2Config struct {
	AccessKeyID     string
	SecretAccessKey string
	BucketName      string
	Endpoint        string
	Region          string
}

func LoadAppConfig() AppConfig {
	return AppConfig{
		Port:          GetEnv("SEFUD_APP_PORT", "7000"),
		StoragePath:   GetEnv("SEFUD_STORAGE_PATH", "./uploads"),
		MimeBlacklist: strings.Split(GetEnv("SEFUD_MIME_BLACKLIST", "application/x-sh"), ","),
		MaxUploadSize: GetEnvAsInt64("SEFUD_MAX_UPLOAD_SIZE", 100*1024*1024), // 100MB default
	}
}

func LoadR2Config() R2Config {
	return R2Config{
		AccessKeyID:     GetEnv("SEFUD_R2_ACCESS_KEY_ID", ""),
		SecretAccessKey: GetEnv("SEFUD_R2_SECRET_ACCESS_KEY", ""),
		BucketName:      GetEnv("SEFUD_R2_BUCKET_NAME", "sefud-files"),
		Endpoint:        GetEnv("SEFUD_R2_ENDPOINT", ""),
		Region:          GetEnv("SEFUD_R2_REGION", "auto"),
	}
}
