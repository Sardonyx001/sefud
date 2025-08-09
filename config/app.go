// Package config provides configuration settings for the application.
package config

import (
	"strings"
)

type AppConfig struct {
	Port          string
	MimeBlacklist []string
	MaxUploadSize int64
}

type StorageConfig struct {
	AccessKeyID     string
	SecretAccessKey string
	BucketName      string
	Endpoint        string
	Region          string
}

func LoadAppConfig() AppConfig {
	return AppConfig{
		Port:          GetEnv("SEFUD_APP_PORT", "7000"),
		MimeBlacklist: strings.Split(GetEnv("SEFUD_APP_MIME_BLACKLIST", "application/x-sh"), ","),
		MaxUploadSize: GetEnvAsInt64("SEFUD_APP_MAX_UPLOAD_SIZE", 100*1024*1024), // 100MB default
	}
}

func LoadStorageConfig() StorageConfig {
	return StorageConfig{
		AccessKeyID:     GetEnv("SEFUD_STORAGE_ACCESS_KEY_ID", ""),
		SecretAccessKey: GetEnv("SEFUD_STORAGE_SECRET_ACCESS_KEY", ""),
		BucketName:      GetEnv("SEFUD_STORAGE_BUCKET_NAME", "sefud-files"),
		Endpoint:        GetEnv("SEFUD_STORAGE_ENDPOINT", ""),
		Region:          GetEnv("SEFUD_STORAGE_REGION", "auto"),
	}
}
