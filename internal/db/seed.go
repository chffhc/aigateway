package db

import (
	"log"
)

// ProviderType represents the type of provider/endpoint
const (
	ProviderTypeOpenAI    = "openai"
	ProviderTypeAnthropic = "anthropic"
)

// ProviderCategory distinguishes between regular API and coding plan
const (
	CategoryAPI       = "api"
	CategoryCodingPlan = "coding_plan"
)

// ProviderPreset holds preset configuration for a provider endpoint
type ProviderPreset struct {
	Name     string
	Type     string // openai, anthropic
	Category string // api, coding_plan
	BaseURL  string
	Remark   string
	Priority int
}

// GetAllPresets returns all preset provider configurations
func GetAllPresets() []ProviderPreset {
	return []ProviderPreset{
		// ===== OpenAI =====
		{
			Name: "openai-official", Type: ProviderTypeOpenAI, Category: CategoryAPI,
			BaseURL: "https://api.openai.com",
			Remark:  "OpenAI 官方 API", Priority: 100,
		},

		// ===== Anthropic =====
		{
			Name: "anthropic-official", Type: ProviderTypeAnthropic, Category: CategoryAPI,
			BaseURL: "https://api.anthropic.com",
			Remark:  "Anthropic 官方 API", Priority: 100,
		},

		// ===== Google Gemini =====
		{
			Name: "google-gemini", Type: ProviderTypeOpenAI, Category: CategoryAPI,
			BaseURL: "https://generativelanguage.googleapis.com/v1beta/openai",
			Remark:  "Google Gemini API (OpenAI 兼容)", Priority: 100,
		},

		// ===== DeepSeek =====
		{
			Name: "deepseek-official", Type: ProviderTypeOpenAI, Category: CategoryAPI,
			BaseURL: "https://api.deepseek.com",
			Remark:  "DeepSeek 官方 API（V4/V3.2）", Priority: 100,
		},

		// ===== 阿里云百炼 - 通用 API =====
		{
			Name: "aliyun-dashscope", Type: ProviderTypeOpenAI, Category: CategoryAPI,
			BaseURL: "https://dashscope.aliyuncs.com/compatible-mode/v1",
			Remark:  "阿里云百炼 DashScope (按量付费)", Priority: 100,
		},
		// ===== 阿里云百炼 - Coding Plan =====
		{
			Name: "aliyun-coding-openai", Type: ProviderTypeOpenAI, Category: CategoryCodingPlan,
			BaseURL: "https://coding.dashscope.aliyuncs.com/v1",
			Remark:  "阿里云百炼 Coding Plan (OpenAI 兼容) | Key 以 sk-sp- 开头", Priority: 100,
		},
		{
			Name: "aliyun-coding-anthropic", Type: ProviderTypeAnthropic, Category: CategoryCodingPlan,
			BaseURL: "https://coding.dashscope.aliyuncs.com/apps/anthropic/v1",
			Remark:  "阿里云百炼 Coding Plan (Anthropic 兼容) | Key 以 sk-sp- 开头", Priority: 100,
		},
		// ===== 阿里云百炼 - Coding Plan 国际版 =====
		{
			Name: "aliyun-coding-intl-openai", Type: ProviderTypeOpenAI, Category: CategoryCodingPlan,
			BaseURL: "https://coding-intl.dashscope.aliyuncs.com/v1",
			Remark:  "阿里云百炼 Coding Plan 国际版 (OpenAI 兼容)", Priority: 90,
		},
		{
			Name: "aliyun-coding-intl-anthropic", Type: ProviderTypeAnthropic, Category: CategoryCodingPlan,
			BaseURL: "https://coding-intl.dashscope.aliyuncs.com/apps/anthropic",
			Remark:  "阿里云百炼 Coding Plan 国际版 (Anthropic 兼容)", Priority: 90,
		},

		// ===== 智谱 AI - 通用 API =====
		{
			Name: "zhipu-official", Type: ProviderTypeOpenAI, Category: CategoryAPI,
			BaseURL: "https://open.bigmodel.cn/api/paas/v4",
			Remark:  "智谱 AI 官方 API (按量付费)", Priority: 100,
		},
		// ===== 智谱 AI - Coding Plan =====
		{
			Name: "zhipu-coding-openai", Type: ProviderTypeOpenAI, Category: CategoryCodingPlan,
			BaseURL: "https://open.bigmodel.cn/api/coding/paas/v4",
			Remark:  "智谱 Coding Plan (OpenAI 兼容) | 专属 Coding Key", Priority: 100,
		},
		{
			Name: "zhipu-coding-anthropic", Type: ProviderTypeAnthropic, Category: CategoryCodingPlan,
			BaseURL: "https://open.bigmodel.cn/api/anthropic",
			Remark:  "智谱 Coding Plan (Anthropic 兼容) | 专属 Coding Key", Priority: 100,
		},

		// ===== Moonshot/Kimi - 通用 API =====
		{
			Name: "moonshot-official", Type: ProviderTypeOpenAI, Category: CategoryAPI,
			BaseURL: "https://api.moonshot.cn/v1",
			Remark:  "Moonshot Kimi 官方 API (按量付费)", Priority: 100,
		},
		// ===== Moonshot/Kimi - Coding Plan =====
		{
			Name: "kimi-coding", Type: ProviderTypeOpenAI, Category: CategoryCodingPlan,
			BaseURL: "https://api.kimi.com/coding/v1",
			Remark:  "Kimi Coding Plan | 专属 Coding Key", Priority: 100,
		},

		// ===== MiniMax - 通用 API =====
		{
			Name: "minimax-official", Type: ProviderTypeOpenAI, Category: CategoryAPI,
			BaseURL: "https://api.minimax.chat/v1",
			Remark:  "MiniMax 官方 API (按量付费)", Priority: 100,
		},
		// ===== MiniMax - Coding Plan (Token Plan) =====
		{
			Name: "minimax-coding", Type: ProviderTypeOpenAI, Category: CategoryCodingPlan,
			BaseURL: "https://api.minimax.chat/v1",
			Remark:  "MiniMax Token Plan (Coding) | 同 Base URL，使用 Token Plan Key", Priority: 100,
		},

		// ===== 小米 AI =====
		{
			Name: "xiaomi-official", Type: ProviderTypeOpenAI, Category: CategoryAPI,
			BaseURL: "https://api.xiaomi.com/v1",
			Remark:  "小米 AI 官方 API", Priority: 100,
		},

		// ===== Groq =====
		{
			Name: "groq-official", Type: ProviderTypeOpenAI, Category: CategoryAPI,
			BaseURL: "https://api.groq.com/openai/v1",
			Remark:  "Groq 高速推理 API", Priority: 90,
		},

		// ===== SiliconFlow =====
		{
			Name: "siliconflow", Type: ProviderTypeOpenAI, Category: CategoryAPI,
			BaseURL: "https://api.siliconflow.cn/v1",
			Remark:  "SiliconFlow 聚合平台", Priority: 90,
		},

		// ===== Together AI =====
		{
			Name: "together-ai", Type: ProviderTypeOpenAI, Category: CategoryAPI,
			BaseURL: "https://api.together.xyz/v1",
			Remark:  "Together AI 聚合平台", Priority: 80,
		},

		// ===== 自定义通用接口 =====
		{
			Name: "custom-openai", Type: ProviderTypeOpenAI, Category: CategoryAPI,
			BaseURL: "https://your-api.example.com/v1",
			Remark:  "自定义 OpenAI 兼容接口", Priority: 50,
		},
		{
			Name: "custom-anthropic", Type: ProviderTypeAnthropic, Category: CategoryAPI,
			BaseURL: "https://your-api.example.com/v1",
			Remark:  "自定义 Anthropic 兼容接口", Priority: 50,
		},
	}
}

// SeedProviders inserts default provider configurations if none exist
func SeedProviders() {
	var count int64
	DB.Model(&ProviderConfig{}).Count(&count)
	if count > 0 {
		log.Printf("Skipping provider seed: %d providers already exist", count)
		return
	}

	enabled := true
	presets := GetAllPresets()

	for _, p := range presets {
		provider := ProviderConfig{
			Name:      p.Name,
			Type:      p.Type,
			Category:  p.Category,
			BaseURL:   p.BaseURL,
			APIKey:    "",
			IsEnabled: &enabled,
			Timeout:   60,
			Priority:  p.Priority,
			Weight:    1,
			Remark:    p.Remark,
		}
		DB.Create(&provider)
	}

	log.Printf("Seeded %d default providers (%d API, %d Coding Plan, %d Custom)",
		len(presets),
		countByCategory(presets, CategoryAPI),
		countByCategory(presets, CategoryCodingPlan),
		countByCategory(presets, "")) // custom counted separately
}

func countByCategory(presets []ProviderPreset, cat string) int {
	count := 0
	for _, p := range presets {
		if cat == "" {
			if p.Priority == 50 {
				count++
			}
		} else if p.Category == cat {
			count++
		}
	}
	return count
}

// SeedModels inserts default model mappings if none exist
func SeedModels() {
	var count int64
	DB.Model(&ModelMapping{}).Count(&count)
	if count > 0 {
		log.Printf("Skipping model seed: %d models already exist", count)
		return
	}

	enabled := true

	// Get provider IDs - only seed models for API category providers
	var openaiID, anthropicID, deepseekID, geminiID, aliyunID, moonshotID, zhipuID, minimaxID, xiaomiID uint
	DB.Model(&ProviderConfig{}).Where("name = ? AND category = ?", "openai-official", CategoryAPI).Pluck("id", &openaiID)
	DB.Model(&ProviderConfig{}).Where("name = ? AND category = ?", "anthropic-official", CategoryAPI).Pluck("id", &anthropicID)
	DB.Model(&ProviderConfig{}).Where("name = ? AND category = ?", "deepseek-official", CategoryAPI).Pluck("id", &deepseekID)
	DB.Model(&ProviderConfig{}).Where("name = ? AND category = ?", "google-gemini", CategoryAPI).Pluck("id", &geminiID)
	DB.Model(&ProviderConfig{}).Where("name = ? AND category = ?", "aliyun-dashscope", CategoryAPI).Pluck("id", &aliyunID)
	DB.Model(&ProviderConfig{}).Where("name = ? AND category = ?", "moonshot-official", CategoryAPI).Pluck("id", &moonshotID)
	DB.Model(&ProviderConfig{}).Where("name = ? AND category = ?", "zhipu-official", CategoryAPI).Pluck("id", &zhipuID)
	DB.Model(&ProviderConfig{}).Where("name = ? AND category = ?", "minimax-official", CategoryAPI).Pluck("id", &minimaxID)
	DB.Model(&ProviderConfig{}).Where("name = ? AND category = ?", "xiaomi-official", CategoryAPI).Pluck("id", &xiaomiID)

	// Also get coding plan provider IDs for model mapping
	var aliyunCodingID, zhipuCodingID, kimiCodingID uint
	DB.Model(&ProviderConfig{}).Where("name = ?", "aliyun-coding-openai").Pluck("id", &aliyunCodingID)
	DB.Model(&ProviderConfig{}).Where("name = ?", "zhipu-coding-openai").Pluck("id", &zhipuCodingID)
	DB.Model(&ProviderConfig{}).Where("name = ?", "kimi-coding").Pluck("id", &kimiCodingID)

	models := []ModelMapping{
		// ===== OpenAI 模型 =====
		// GPT-5.5 系列（2026-04-24 最新发布）
		{Name: "gpt-5.5", ProviderConfigID: openaiID, UpstreamModel: "gpt-5.5", IsEnabled: &enabled, Weight: 1},
		{Name: "gpt-5.5-pro", ProviderConfigID: openaiID, UpstreamModel: "gpt-5.5-pro", IsEnabled: &enabled, Weight: 1},
		{Name: "gpt-5.5-mini", ProviderConfigID: openaiID, UpstreamModel: "gpt-5.5-mini", IsEnabled: &enabled, Weight: 1},
		{Name: "gpt-5.4", ProviderConfigID: openaiID, UpstreamModel: "gpt-5.4", IsEnabled: &enabled, Weight: 1},
		{Name: "gpt-5.4-mini", ProviderConfigID: openaiID, UpstreamModel: "gpt-5.4-mini", IsEnabled: &enabled, Weight: 1},
		{Name: "gpt-5.2", ProviderConfigID: openaiID, UpstreamModel: "gpt-5.2", IsEnabled: &enabled, Weight: 1},
		{Name: "gpt-5.2-mini", ProviderConfigID: openaiID, UpstreamModel: "gpt-5.2-mini", IsEnabled: &enabled, Weight: 1},
		{Name: "o3", ProviderConfigID: openaiID, UpstreamModel: "o3", IsEnabled: &enabled, Weight: 1},
		{Name: "o4-mini", ProviderConfigID: openaiID, UpstreamModel: "o4-mini", IsEnabled: &enabled, Weight: 1},
		{Name: "gpt-oss-120b", ProviderConfigID: openaiID, UpstreamModel: "gpt-oss-120b", IsEnabled: &enabled, Weight: 1},
		{Name: "gpt-oss-20b", ProviderConfigID: openaiID, UpstreamModel: "gpt-oss-20b", IsEnabled: &enabled, Weight: 1},

		// ===== Anthropic Claude 模型 =====
		{Name: "claude-sonnet-4-6", ProviderConfigID: anthropicID, UpstreamModel: "claude-sonnet-4-6", IsEnabled: &enabled, Weight: 1},
		{Name: "claude-opus-4-6", ProviderConfigID: anthropicID, UpstreamModel: "claude-opus-4-6", IsEnabled: &enabled, Weight: 1},
		{Name: "claude-opus-4-7", ProviderConfigID: anthropicID, UpstreamModel: "claude-opus-4-7", IsEnabled: &enabled, Weight: 1},
		{Name: "claude-haiku-4-5", ProviderConfigID: anthropicID, UpstreamModel: "claude-haiku-4-5", IsEnabled: &enabled, Weight: 1},

		// ===== Google Gemini 模型 =====
		{Name: "gemini-3.1-pro", ProviderConfigID: geminiID, UpstreamModel: "gemini-3.1-pro-preview", IsEnabled: &enabled, Weight: 1},
		{Name: "gemini-3.1-flash", ProviderConfigID: geminiID, UpstreamModel: "gemini-3.1-flash-preview", IsEnabled: &enabled, Weight: 1},
		{Name: "gemini-3.1-flash-lite", ProviderConfigID: geminiID, UpstreamModel: "gemini-3.1-flash-lite-preview", IsEnabled: &enabled, Weight: 1},
		{Name: "gemini-3-flash", ProviderConfigID: geminiID, UpstreamModel: "gemini-3-flash-preview", IsEnabled: &enabled, Weight: 1},
		{Name: "gemini-2.5-pro", ProviderConfigID: geminiID, UpstreamModel: "gemini-2.5-pro", IsEnabled: &enabled, Weight: 1},
		{Name: "gemini-2.5-flash", ProviderConfigID: geminiID, UpstreamModel: "gemini-2.5-flash", IsEnabled: &enabled, Weight: 1},
		{Name: "gemini-2.5-flash-lite", ProviderConfigID: geminiID, UpstreamModel: "gemini-2.5-flash-lite", IsEnabled: &enabled, Weight: 1},

		// ===== DeepSeek 模型 =====
		// V4 系列（2026-04-24 最新发布）
		{Name: "deepseek-v4-pro", ProviderConfigID: deepseekID, UpstreamModel: "deepseek-v4-pro", IsEnabled: &enabled, Weight: 1},
		{Name: "deepseek-v4-flash", ProviderConfigID: deepseekID, UpstreamModel: "deepseek-v4-flash", IsEnabled: &enabled, Weight: 1},
		// V3.2 系列（2026-07-24 废弃）
		{Name: "deepseek-v3.2", ProviderConfigID: deepseekID, UpstreamModel: "deepseek-v3.2", IsEnabled: &enabled, Weight: 1},
		{Name: "deepseek-v3.2-speciale", ProviderConfigID: deepseekID, UpstreamModel: "deepseek-v3.2-speciale", IsEnabled: &enabled, Weight: 1},
		// 兼容别名（即将废弃，映射到 V4 Flash）
		{Name: "deepseek-chat", ProviderConfigID: deepseekID, UpstreamModel: "deepseek-v4-flash", IsEnabled: &enabled, Weight: 1},
		{Name: "deepseek-reasoner", ProviderConfigID: deepseekID, UpstreamModel: "deepseek-v4-flash", IsEnabled: &enabled, Weight: 1},

		// ===== 阿里云百炼 Qwen 模型 (通用 API) =====
		{Name: "qwen3.6-max-preview", ProviderConfigID: aliyunID, UpstreamModel: "qwen3.6-max-preview", IsEnabled: &enabled, Weight: 1},
		{Name: "qwen3.6-plus", ProviderConfigID: aliyunID, UpstreamModel: "qwen3.6-plus", IsEnabled: &enabled, Weight: 1},
		{Name: "qwen3.5-plus", ProviderConfigID: aliyunID, UpstreamModel: "qwen3.5-plus", IsEnabled: &enabled, Weight: 1},
		{Name: "qwen3-max", ProviderConfigID: aliyunID, UpstreamModel: "qwen3-max", IsEnabled: &enabled, Weight: 1},
		{Name: "qwen3-max-thinking", ProviderConfigID: aliyunID, UpstreamModel: "qwen3-max-thinking", IsEnabled: &enabled, Weight: 1},
		{Name: "qwen3-plus", ProviderConfigID: aliyunID, UpstreamModel: "qwen3-plus", IsEnabled: &enabled, Weight: 1},
		{Name: "qwen3-flash", ProviderConfigID: aliyunID, UpstreamModel: "qwen3-flash", IsEnabled: &enabled, Weight: 1},
		{Name: "qwen3.6-flash", ProviderConfigID: aliyunID, UpstreamModel: "qwen3.6-flash", IsEnabled: &enabled, Weight: 1},
		{Name: "qwen3-coder", ProviderConfigID: aliyunID, UpstreamModel: "qwen3-coder", IsEnabled: &enabled, Weight: 1},

		// ===== 阿里云百炼 Coding Plan 模型 =====
		{Name: "qwen3-max", ProviderConfigID: aliyunCodingID, UpstreamModel: "qwen3-max", IsEnabled: &enabled, Weight: 1},
		{Name: "qwen3.6-plus", ProviderConfigID: aliyunCodingID, UpstreamModel: "qwen3.6-plus", IsEnabled: &enabled, Weight: 1},
		{Name: "qwen3-coder", ProviderConfigID: aliyunCodingID, UpstreamModel: "qwen3-coder", IsEnabled: &enabled, Weight: 1},

		// ===== Moonshot Kimi 模型 (通用 API) =====
		{Name: "kimi-k2", ProviderConfigID: moonshotID, UpstreamModel: "kimi-k2", IsEnabled: &enabled, Weight: 1},
		{Name: "kimi-k2-thinking", ProviderConfigID: moonshotID, UpstreamModel: "kimi-k2-thinking", IsEnabled: &enabled, Weight: 1},
		{Name: "kimi-k2-6", ProviderConfigID: moonshotID, UpstreamModel: "kimi-k2-6", IsEnabled: &enabled, Weight: 1},

		// ===== Kimi Coding Plan 模型 =====
		{Name: "kimi-for-coding", ProviderConfigID: kimiCodingID, UpstreamModel: "kimi-for-coding", IsEnabled: &enabled, Weight: 1},
		{Name: "kimi-k2-6", ProviderConfigID: kimiCodingID, UpstreamModel: "kimi-k2-6", IsEnabled: &enabled, Weight: 1},

		// ===== 智谱 GLM 模型 (通用 API) =====
		{Name: "glm-5", ProviderConfigID: zhipuID, UpstreamModel: "glm-5", IsEnabled: &enabled, Weight: 1},
		{Name: "glm-5-flash", ProviderConfigID: zhipuID, UpstreamModel: "glm-5-flash", IsEnabled: &enabled, Weight: 1},
		{Name: "glm-4-plus", ProviderConfigID: zhipuID, UpstreamModel: "glm-4-plus", IsEnabled: &enabled, Weight: 1},
		{Name: "glm-4-flash", ProviderConfigID: zhipuID, UpstreamModel: "glm-4-flash", IsEnabled: &enabled, Weight: 1},
		{Name: "glm-4v", ProviderConfigID: zhipuID, UpstreamModel: "glm-4v", IsEnabled: &enabled, Weight: 1},

		// ===== 智谱 Coding Plan 模型 =====
		{Name: "glm-5", ProviderConfigID: zhipuCodingID, UpstreamModel: "glm-5", IsEnabled: &enabled, Weight: 1},
		{Name: "glm-5-flash", ProviderConfigID: zhipuCodingID, UpstreamModel: "glm-5-flash", IsEnabled: &enabled, Weight: 1},
		{Name: "glm-4-plus", ProviderConfigID: zhipuCodingID, UpstreamModel: "glm-4-plus", IsEnabled: &enabled, Weight: 1},

		// ===== MiniMax 模型 =====
		{Name: "minimax-m2.7", ProviderConfigID: minimaxID, UpstreamModel: "minimax-m2.7", IsEnabled: &enabled, Weight: 1},
		{Name: "minimax-m2.7-highspeed", ProviderConfigID: minimaxID, UpstreamModel: "minimax-m2.7-highspeed", IsEnabled: &enabled, Weight: 1},
		{Name: "minimax-m2.5", ProviderConfigID: minimaxID, UpstreamModel: "minimax-m2.5", IsEnabled: &enabled, Weight: 1},
		{Name: "minimax-m2.5-highspeed", ProviderConfigID: minimaxID, UpstreamModel: "minimax-m2.5-highspeed", IsEnabled: &enabled, Weight: 1},
		{Name: "minimax-m2", ProviderConfigID: minimaxID, UpstreamModel: "minimax-m2", IsEnabled: &enabled, Weight: 1},
		{Name: "minimax-m2-her", ProviderConfigID: minimaxID, UpstreamModel: "minimax-m2-her", IsEnabled: &enabled, Weight: 1},

		// ===== 小米 AI =====
		{Name: "mimo", ProviderConfigID: xiaomiID, UpstreamModel: "mimo", IsEnabled: &enabled, Weight: 1},
		{Name: "mimo-v2", ProviderConfigID: xiaomiID, UpstreamModel: "mimo-v2", IsEnabled: &enabled, Weight: 1},
	}

	for _, m := range models {
		DB.Create(&m)
	}

	log.Printf("Seeded %d default models", len(models))
}

// SeedPrices inserts default model pricing if none exist
func SeedPrices() {
	var count int64
	DB.Model(&ModelPrice{}).Count(&count)
	if count > 0 {
		log.Printf("Skipping price seed: %d prices already exist", count)
		return
	}

	// Pricing in USD per 1M tokens (approximate as of April 2026)
	prices := []ModelPrice{
		// OpenAI GPT-5.5 series
		{Model: "gpt-5.5", InputPerMillion: 2.50, OutputPerMillion: 15.00},
		{Model: "gpt-5.5-pro", InputPerMillion: 10.00, OutputPerMillion: 60.00},
		{Model: "gpt-5.5-mini", InputPerMillion: 0.25, OutputPerMillion: 1.50},
		{Model: "gpt-5.4", InputPerMillion: 1.25, OutputPerMillion: 10.00},
		{Model: "gpt-5.4-mini", InputPerMillion: 0.15, OutputPerMillion: 0.60},
		{Model: "o3", InputPerMillion: 10.00, OutputPerMillion: 40.00},
		{Model: "o4-mini", InputPerMillion: 1.10, OutputPerMillion: 4.40},

		// Anthropic Claude 4.x
		{Model: "claude-sonnet-4-6", InputPerMillion: 3.00, OutputPerMillion: 15.00},
		{Model: "claude-opus-4-6", InputPerMillion: 15.00, OutputPerMillion: 75.00},
		{Model: "claude-opus-4-7", InputPerMillion: 15.00, OutputPerMillion: 75.00},
		{Model: "claude-haiku-4-5", InputPerMillion: 0.80, OutputPerMillion: 4.00},

		// Google Gemini
		{Model: "gemini-3.1-pro", InputPerMillion: 1.25, OutputPerMillion: 10.00},
		{Model: "gemini-3.1-flash", InputPerMillion: 0.15, OutputPerMillion: 3.50},
		{Model: "gemini-3.1-flash-lite", InputPerMillion: 0.075, OutputPerMillion: 0.30},
		{Model: "gemini-2.5-pro", InputPerMillion: 1.25, OutputPerMillion: 10.00},
		{Model: "gemini-2.5-flash", InputPerMillion: 0.30, OutputPerMillion: 2.50},

		// DeepSeek V4 系列
		{Model: "deepseek-v4-pro", InputPerMillion: 1.74, OutputPerMillion: 3.48},
		{Model: "deepseek-v4-flash", InputPerMillion: 0.14, OutputPerMillion: 0.28},
		// DeepSeek V3.2 系列
		{Model: "deepseek-v3.2", InputPerMillion: 0.27, OutputPerMillion: 1.10},
		// 兼容别名（映射到 V4 Flash）
		{Model: "deepseek-chat", InputPerMillion: 0.14, OutputPerMillion: 0.28},
		{Model: "deepseek-reasoner", InputPerMillion: 0.14, OutputPerMillion: 0.28},

		// 阿里云百炼 Qwen
		{Model: "qwen3.6-max-preview", InputPerMillion: 1.60, OutputPerMillion: 6.40},
		{Model: "qwen3.6-plus", InputPerMillion: 0.40, OutputPerMillion: 1.20},
		{Model: "qwen3-max", InputPerMillion: 1.60, OutputPerMillion: 6.40},
		{Model: "qwen3-flash", InputPerMillion: 0.05, OutputPerMillion: 0.20},
		{Model: "qwen3.6-flash", InputPerMillion: 0.05, OutputPerMillion: 0.20},
		{Model: "qwen3-coder", InputPerMillion: 0.80, OutputPerMillion: 3.20},

		// Moonshot Kimi
		{Model: "kimi-k2", InputPerMillion: 0.50, OutputPerMillion: 2.00},
		{Model: "kimi-k2-6", InputPerMillion: 0.50, OutputPerMillion: 2.00},
		{Model: "kimi-for-coding", InputPerMillion: 0.30, OutputPerMillion: 1.20},

		// 智谱 GLM
		{Model: "glm-5", InputPerMillion: 0.50, OutputPerMillion: 2.00},
		{Model: "glm-5-flash", InputPerMillion: 0.10, OutputPerMillion: 0.40},
		{Model: "glm-4-plus", InputPerMillion: 0.50, OutputPerMillion: 2.00},

		// MiniMax
		{Model: "minimax-m2.7", InputPerMillion: 0.50, OutputPerMillion: 2.00},
		{Model: "minimax-m2.5", InputPerMillion: 0.50, OutputPerMillion: 2.00},
		{Model: "minimax-m2", InputPerMillion: 0.50, OutputPerMillion: 2.00},

		// 小米
		{Model: "mimo", InputPerMillion: 0.30, OutputPerMillion: 1.20},
		{Model: "mimo-v2", InputPerMillion: 0.30, OutputPerMillion: 1.20},
	}

	for _, p := range prices {
		DB.Create(&p)
	}

	log.Printf("Seeded %d default model prices", len(prices))
}
