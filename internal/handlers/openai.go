package handlers

import (
	"encoding/json"
	"io"
	"math"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vicecatcher/aigateway/internal/db"
	"github.com/vicecatcher/aigateway/internal/middleware"
	"github.com/vicecatcher/aigateway/internal/proxy"
)

func RegisterOpenAI(g *gin.Engine, p *proxy.Proxy) {
	v1 := g.Group("/v1")

	// Public: list available models
	v1.GET("/models", func(c *gin.Context) {
		var mappings []db.ModelMapping
		db.DB.Where("is_enabled = 1").Find(&mappings)

		models := make([]gin.H, 0, len(mappings))
		for _, m := range mappings {
			models = append(models, gin.H{
				"id":       m.Name,
				"object":   "model",
				"created":  m.CreatedAt.Unix(),
				"owned_by": m.ProviderConfig.Type,
			})
		}

		c.JSON(200, gin.H{
			"object": "list",
			"data":   models,
		})
	})

	// Authenticated: chat completions
	v1.POST("/chat/completions",
		middleware.APIKeyAuth(),
		openAIChatHandler(p),
	)

	v1.POST("/completions",
		middleware.APIKeyAuth(),
		openAIChatHandler(p),
	)
}

func openAIChatHandler(p *proxy.Proxy) gin.HandlerFunc {
	return func(c *gin.Context) {
		userCtx, ok := middleware.GetUserContext(c)
		if !ok {
			return
		}

		if !checkQuota(c, userCtx.UserID) {
			return
		}

		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(400, gin.H{"error": "failed to read body"})
			return
		}

		var req map[string]interface{}
		if err := json.Unmarshal(body, &req); err != nil {
			c.JSON(400, gin.H{"error": "invalid JSON"})
			return
		}

		model, ok := req["model"].(string)
		if !ok || model == "" {
			c.JSON(400, gin.H{"error": "model is required"})
			return
		}

		upstream, err := proxy.ResolveUpstream(model)
		if err != nil {
			c.JSON(404, gin.H{"error": err.Error()})
			return
		}

		upstream.Body = body

		if stream, _ := req["stream"].(bool); stream {
			p.HandleOpenAIStream(c, userCtx, *upstream)
			return
		}

		p.HandleOpenAI(c, userCtx, *upstream)
	}
}

func checkQuota(c *gin.Context, userID uint) bool {
	var user db.User
	if err := db.DB.First(&user, userID).Error; err != nil {
		c.JSON(500, gin.H{"error": "user not found"})
		return false
	}

	if user.QuotaType == "token" && user.TokenQuota > 0 {
		if user.TokenUsed >= user.TokenQuota {
			c.JSON(402, gin.H{
				"error":      "quota exceeded",
				"quota_type": "token",
				"used":       user.TokenUsed,
				"total":      user.TokenQuota,
			})
			return false
		}
	} else if user.QuotaType == "calls" && user.CallQuota > 0 {
		if user.CallUsed >= user.CallQuota {
			c.JSON(402, gin.H{
				"error":      "quota exceeded",
				"quota_type": "calls",
				"used":       user.CallUsed,
				"total":      user.CallQuota,
			})
			return false
		}
	}

	return true
}

func extractModelFromBody(body []byte) string {
	var req map[string]interface{}
	if err := json.Unmarshal(body, &req); err != nil {
		return ""
	}
	if model, ok := req["model"].(string); ok {
		return model
	}
	return ""
}

func isAnthropicType(providerType string) bool {
	return strings.ToLower(providerType) == "anthropic"
}

// RegisterAPIKeyStats registers API key usage statistics endpoints
func RegisterAPIKeyStats(g *gin.Engine) {
	api := g.Group("/api/stats")
	api.Use(middleware.APIKeyAuth())

	// Per-key usage stats
	api.GET("/keys", apikeyStats)
	
	// Per-model usage stats (for admin)
	api.GET("/models", modelStats)

	// Cost estimation
	api.GET("/cost", costEstimation)
}

func apikeyStats(c *gin.Context) {
	userCtx, _ := middleware.GetUserContext(c)
	
	// Time range
	startDate := c.DefaultQuery("start_date", time.Now().AddDate(0, 0, -30).Format("2006-01-02"))
	endDate := c.DefaultQuery("end_date", time.Now().Format("2006-01-02"))

	// Get all keys for this user
	var keys []db.APIKey
	db.DB.Where("user_id = ?", userCtx.UserID).Find(&keys)

	type KeyStats struct {
		KeyID       uint    `json:"key_id"`
		KeyName     string  `json:"key_name"`
		KeyPrefix   string  `json:"key_prefix"`
		CallCount   int64   `json:"call_count"`
		TokenCount  int64   `json:"token_count"`
		InputTokens int64   `json:"input_tokens"`
		OutputTokens int64  `json:"output_tokens"`
		AvgLatency  float64 `json:"avg_latency_ms"`
		SuccessRate float64 `json:"success_rate"`
		LastUsed    *time.Time `json:"last_used_at"`
	}

	stats := make([]KeyStats, 0, len(keys))
	for _, key := range keys {
		var ks KeyStats
		ks.KeyID = key.ID
		ks.KeyName = key.Name
		if len(key.Key) > 12 {
			ks.KeyPrefix = key.Key[:12] + "..."
		} else {
			ks.KeyPrefix = key.Key
		}

		// Query usage logs for this key
		db.DB.Model(&db.UsageLog{}).
			Where("api_key_id = ? AND created_at >= ? AND created_at <= ?", key.ID, startDate+" 00:00:00", endDate+" 23:59:59").
			Select("COUNT(*) as call_count, COALESCE(SUM(total_tokens), 0) as token_count, COALESCE(SUM(input_tokens), 0) as input_tokens, COALESCE(SUM(output_tokens), 0) as output_tokens, AVG(latency_ms) as avg_latency").
			Scan(&struct {
				CallCount   *int64
				TokenCount  *int64
				InputTokens *int64
				OutputTokens *int64
				AvgLatency  *float64
			}{
				&ks.CallCount, &ks.TokenCount, &ks.InputTokens, &ks.OutputTokens, &ks.AvgLatency,
			})

		// Success rate
		var successCount int64
		db.DB.Model(&db.UsageLog{}).
			Where("api_key_id = ? AND created_at >= ? AND created_at <= ? AND status_code < 400", key.ID, startDate+" 00:00:00", endDate+" 23:59:59").
			Count(&successCount)
		if ks.CallCount > 0 {
			ks.SuccessRate = float64(successCount) / float64(ks.CallCount) * 100
		}

		// Last used
		var lastUsed *time.Time
		db.DB.Model(&db.UsageLog{}).
			Where("api_key_id = ?", key.ID).
			Order("created_at DESC").
			Pluck("created_at", &lastUsed)
		ks.LastUsed = lastUsed

		stats = append(stats, ks)
	}

	c.JSON(200, gin.H{
		"data":       stats,
		"start_date": startDate,
		"end_date":   endDate,
	})
}

func modelStats(c *gin.Context) {
	userCtx, _ := middleware.GetUserContext(c)

	startDate := c.DefaultQuery("start_date", time.Now().AddDate(0, 0, -30).Format("2006-01-02"))
	endDate := c.DefaultQuery("end_date", time.Now().Format("2006-01-02"))

	type ModelStat struct {
		Model        string  `json:"model"`
		ProviderType string  `json:"provider_type"`
		CallCount    int64   `json:"call_count"`
		TokenCount   int64   `json:"token_count"`
		InputTokens  int64   `json:"input_tokens"`
		OutputTokens int64   `json:"output_tokens"`
		AvgLatency   float64 `json:"avg_latency_ms"`
	}

	var stats []ModelStat
	db.DB.Model(&db.UsageLog{}).
		Where("user_id = ? AND created_at >= ? AND created_at <= ?", userCtx.UserID, startDate+" 00:00:00", endDate+" 23:59:59").
		Select("model, provider_type, COUNT(*) as call_count, COALESCE(SUM(total_tokens), 0) as token_count, COALESCE(SUM(input_tokens), 0) as input_tokens, COALESCE(SUM(output_tokens), 0) as output_tokens, AVG(latency_ms) as avg_latency").
		Group("model, provider_type").
		Order("call_count DESC").
		Scan(&stats)

	c.JSON(200, gin.H{
		"data":       stats,
		"start_date": startDate,
		"end_date":   endDate,
	})
}

func costEstimation(c *gin.Context) {
	userCtx, _ := middleware.GetUserContext(c)

	startDate := c.DefaultQuery("start_date", time.Now().AddDate(0, 0, -30).Format("2006-01-02"))
	endDate := c.DefaultQuery("end_date", time.Now().Format("2006-01-02"))

	// Get all usage logs for this user in the time range
	var logs []db.UsageLog
	db.DB.Where("user_id = ? AND created_at >= ? AND created_at <= ?", userCtx.UserID, startDate+" 00:00:00", endDate+" 23:59:59").Find(&logs)

	// Get pricing for models
	var prices []db.ModelPrice
	db.DB.Find(&prices)
	priceMap := make(map[string]db.ModelPrice)
	for _, p := range prices {
		priceMap[p.Model] = p
	}

	type CostDetail struct {
		Model        string  `json:"model"`
		Calls        int     `json:"calls"`
		InputTokens  int     `json:"input_tokens"`
		OutputTokens int     `json:"output_tokens"`
		InputCost    float64 `json:"input_cost_usd"`
		OutputCost   float64 `json:"output_cost_usd"`
		TotalCost    float64 `json:"total_cost_usd"`
	}

	costDetails := make(map[string]*CostDetail)
	totalCost := 0.0

	for _, log := range logs {
		if _, exists := costDetails[log.Model]; !exists {
			costDetails[log.Model] = &CostDetail{Model: log.Model}
		}
		cd := costDetails[log.Model]
		cd.Calls++
		cd.InputTokens += log.InputTokens
		cd.OutputTokens += log.OutputTokens

		// Calculate cost
		price, hasPrice := priceMap[log.Model]
		if !hasPrice {
			// Try to find partial match
			for modelName, p := range priceMap {
				if strings.Contains(log.Model, modelName) || strings.Contains(modelName, log.Model) {
					price = p
					hasPrice = true
					break
				}
			}
		}

		if hasPrice {
			inputCost := float64(log.InputTokens) / 1_000_000 * price.InputPerMillion
			outputCost := float64(log.OutputTokens) / 1_000_000 * price.OutputPerMillion
			cd.InputCost += inputCost
			cd.OutputCost += outputCost
			cd.TotalCost = cd.InputCost + cd.OutputCost
		}
	}

	details := make([]CostDetail, 0, len(costDetails))
	for _, cd := range costDetails {
		details = append(details, *cd)
		totalCost += cd.TotalCost
	}

	c.JSON(200, gin.H{
		"data": gin.H{
			"details":     details,
			"total_cost":  math.Round(totalCost*10000) / 10000,
			"start_date":  startDate,
			"end_date":    endDate,
		},
	})
}
