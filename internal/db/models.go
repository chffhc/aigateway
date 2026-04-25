package db

import (
	"time"

	"gorm.io/gorm"
)

// User represents a tenant or admin
type User struct {
	ID                uint           `gorm:"primarykey" json:"id"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`
	Username          string         `gorm:"uniqueIndex;size:64" json:"username"`
	PasswordHash      string         `gorm:"size:128" json:"-"`
	Role              string         `gorm:"size:16;default:tenant" json:"role"` // admin, tenant
	QuotaType         string         `gorm:"size:16;default:token" json:"quota_type"` // token, calls
	TokenQuota        int64          `gorm:"default:0" json:"token_quota"`       // total token quota (0 = unlimited)
	TokenUsed         int64          `gorm:"default:0" json:"token_used"`
	CallQuota         int64          `gorm:"default:0" json:"call_quota"`        // total call quota (0 = unlimited)
	CallUsed          int64          `gorm:"default:0" json:"call_used"`
	Email             string         `gorm:"size:128" json:"email"`
	Remark            string         `gorm:"size:256" json:"remark"`
	IsEnabled         *bool          `gorm:"default:true" json:"is_enabled"`
}

// APIKey represents an API key owned by a user
type APIKey struct {
	ID          uint           `gorm:"primarykey" json:"id"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
	Key         string         `gorm:"uniqueIndex;size:128" json:"key"`
	UserID      uint           `gorm:"index" json:"user_id"`
	User        User           `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Name        string         `gorm:"size:64" json:"name"`
	IsEnabled   *bool          `gorm:"default:true" json:"is_enabled"`
	LastUsedAt  *time.Time     `json:"last_used_at"`
	ExpiredAt   *time.Time     `json:"expired_at"`
}

// ProviderConfig stores provider connection settings
type ProviderConfig struct {
	ID           uint      `gorm:"primarykey" json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Name         string    `gorm:"uniqueIndex;size:64" json:"name"` // e.g. "openai-main"
	Type         string    `gorm:"size:32" json:"type"`             // openai, anthropic, google, etc.
	Category     string    `gorm:"size:16;default:api" json:"category"` // api, coding_plan
	BaseURL      string    `gorm:"size:256" json:"base_url"`
	APIKey       string    `gorm:"size:256" json:"-"` // encrypted or plain
	IsEnabled    *bool     `gorm:"default:true" json:"is_enabled"`
	Timeout      int       `gorm:"default:60" json:"timeout"` // seconds
	Priority     int       `gorm:"default:0" json:"priority"` // load balancing priority
	Weight       int       `gorm:"default:1" json:"weight"`   // load balancing weight
	Remark       string    `gorm:"size:256" json:"remark"`
}

// ModelMapping maps a model name to a provider
type ModelMapping struct {
	ID             uint           `gorm:"primarykey" json:"id"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	Name           string         `gorm:"uniqueIndex:idx_model_provider;size:128" json:"name"` // external model name
	ProviderConfigID uint         `gorm:"uniqueIndex:idx_model_provider;index" json:"provider_config_id"`
	ProviderConfig ProviderConfig `gorm:"foreignKey:ProviderConfigID" json:"provider_config,omitempty"`
	UpstreamModel  string         `gorm:"size:128" json:"upstream_model"` // model name sent to provider (empty = use Name)
	IsEnabled      *bool          `gorm:"default:true" json:"is_enabled"`
	Weight         int            `gorm:"default:1" json:"weight"` // load balancing weight
}

// UsageLog records each API call
type UsageLog struct {
	ID             uint      `gorm:"primarykey" json:"id"`
	CreatedAt      time.Time `json:"created_at"`
	UserID         uint      `gorm:"index" json:"user_id"`
	APIKeyID       uint      `json:"api_key_id"`
	Model          string    `gorm:"size:128" json:"model"`
	ProviderType   string    `gorm:"size:32" json:"provider_type"`
	InputTokens    int       `json:"input_tokens"`
	OutputTokens   int       `json:"output_tokens"`
	TotalTokens    int       `json:"total_tokens"`
	StatusCode     int       `json:"status_code"`
	LatencyMs      int64     `json:"latency_ms"`
}

// DailyUsage aggregates usage per user per day
type DailyUsage struct {
	ID           uint      `gorm:"primarykey" json:"id"`
	UserID       uint      `gorm:"uniqueIndex:idx_user_date" json:"user_id"`
	Date         string    `gorm:"size:10;uniqueIndex:idx_user_date" json:"date"` // YYYY-MM-DD
	CallCount    int64     `gorm:"default:0" json:"call_count"`
	TokenCount   int64     `gorm:"default:0" json:"token_count"`
}

// ModelPrice stores pricing info for cost estimation (optional)
type ModelPrice struct {
	ID              uint      `gorm:"primarykey" json:"id"`
	Model           string    `gorm:"uniqueIndex;size:128" json:"model"`
	InputPerMillion  float64  `json:"input_per_million"`  // USD per 1M input tokens
	OutputPerMillion float64  `json:"output_per_million"` // USD per 1M output tokens
}
