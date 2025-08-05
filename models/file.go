// Package models contains database models for the sefud application.
package models

import (
	"time"

	"gorm.io/gorm"
)

// File represents a file stored in R2 with metadata in the database
type File struct {
	// Primary key - UUID as string
	ID string `gorm:"type:uuid;primary_key" json:"id"`

	// File metadata
	OriginalName string `gorm:"not null" json:"original_name"`
	ContentType  string `gorm:"not null" json:"content_type"`
	Size         int64  `gorm:"not null" json:"size"`

	// R2 storage information
	R2Key    string `gorm:"not null;unique" json:"r2_key"`
	R2Bucket string `gorm:"not null" json:"r2_bucket"`

	// Security and access
	DeleteToken string     `gorm:"not null;unique" json:"delete_token"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`

	// Upload metadata
	UploadIP  string `gorm:"size:45" json:"upload_ip"` // IPv6 compatible
	UserAgent string `json:"user_agent"`

	// Checksums for integrity
	MD5Hash    string `gorm:"size:32" json:"md5_hash"`
	SHA256Hash string `gorm:"size:64" json:"sha256_hash"`

	// Performance tracking
	UploadDuration time.Duration `json:"upload_duration"`

	// Standard timestamps
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

// TableName returns the table name for File model
func (File) TableName() string {
	return "files"
}

// IsExpired checks if the file has expired
func (f *File) IsExpired() bool {
	if f.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*f.ExpiresAt)
}

// GetPublicURL returns the public access URL for the file
func (f *File) GetPublicURL(baseURL string) string {
	return baseURL + "/" + f.ID
}
