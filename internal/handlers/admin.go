package handlers

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vicecatcher/aigateway/internal/db"
	"github.com/vicecatcher/aigateway/internal/middleware"
	"golang.org/x/crypto/bcrypt"
)

func RegisterAdmin(g *gin.Engine) {
	api := g.Group("/api/admin")
	api.Use(middleware.APIKeyAuth(), middleware.RequireRole(middleware.RoleAdmin))

	// User management
	api.GET("/users", adminListUsers)
	api.POST("/users", adminCreateUser)
	api.PUT("/users/:id", adminUpdateUser)
	api.DELETE("/users/:id", adminDeleteUser)
	api.PUT("/users/:id/password", adminChangeUserPassword)

	// API key management (admin can manage all keys)
	api.GET("/keys", adminListKeys)
	api.POST("/keys", adminCreateKey)
	api.DELETE("/keys/:id", adminDeleteKey)

	// Provider management
	api.GET("/providers", adminListProviders)
	api.POST("/providers", adminCreateProvider)
	api.PUT("/providers/:id", adminUpdateProvider)
	api.DELETE("/providers/:id", adminDeleteProvider)
	api.GET("/provider-presets", adminListProviderPresets)

	// Model mapping
	api.GET("/models", adminListModels)
	api.POST("/models", adminCreateModel)
	api.PUT("/models/:id", adminUpdateModel)
	api.DELETE("/models/:id", adminDeleteModel)

	// Global stats
	api.GET("/stats", adminGetStats)
	api.GET("/usage", adminGetUsage)
	
	// Model pricing management
	api.GET("/prices", adminListPrices)
	api.POST("/prices", adminCreatePrice)
	api.PUT("/prices/:id", adminUpdatePrice)
	api.DELETE("/prices/:id", adminDeletePrice)
	
	// Load balancing config
	api.GET("/lb/models/:name", adminGetLBConfig)
	api.PUT("/lb/models/:name/strategy", adminUpdateLBStrategy)
}

func adminListUsers(c *gin.Context) {
	var users []db.User
	db.DB.Order("id desc").Find(&users)
	c.JSON(200, gin.H{"data": users})
}

func adminCreateUser(c *gin.Context) {
	var req struct {
		Username  string `json:"username" binding:"required"`
		Password  string `json:"password" binding:"required"`
		Role      string `json:"role"`
		QuotaType string `json:"quota_type"`
		TokenQuota int64 `json:"token_quota"`
		CallQuota  int64 `json:"call_quota"`
		Email      string `json:"email"`
		Remark     string `json:"remark"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(500, gin.H{"error": "failed to hash password"})
		return
	}

	enabled := true
	user := db.User{
		Username:   req.Username,
		PasswordHash: string(hash),
		Role:       req.Role,
		QuotaType:  req.QuotaType,
		TokenQuota: req.TokenQuota,
		CallQuota:  req.CallQuota,
		Email:      req.Email,
		Remark:     req.Remark,
		IsEnabled:  &enabled,
	}

	if user.Role == "" {
		user.Role = "tenant"
	}
	if user.QuotaType == "" {
		user.QuotaType = "token"
	}

	db.DB.Create(&user)
	c.JSON(201, gin.H{"data": user})
}

func adminUpdateUser(c *gin.Context) {
	id := c.Param("id")
	var user db.User
	if err := db.DB.First(&user, id).Error; err != nil {
		c.JSON(404, gin.H{"error": "user not found"})
		return
	}

	var req struct {
		Role       string `json:"role"`
		QuotaType  string `json:"quota_type"`
		TokenQuota *int64 `json:"token_quota"`
		CallQuota  *int64 `json:"call_quota"`
		IsEnabled  *bool  `json:"is_enabled"`
		Email      string `json:"email"`
		Remark     string `json:"remark"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]interface{}{}
	if req.Role != "" {
		updates["role"] = req.Role
	}
	if req.QuotaType != "" {
		updates["quota_type"] = req.QuotaType
	}
	if req.TokenQuota != nil {
		updates["token_quota"] = *req.TokenQuota
	}
	if req.CallQuota != nil {
		updates["call_quota"] = *req.CallQuota
	}
	if req.IsEnabled != nil {
		updates["is_enabled"] = *req.IsEnabled
	}
	if req.Email != "" {
		updates["email"] = req.Email
	}
	if req.Remark != "" {
		updates["remark"] = req.Remark
	}

	db.DB.Model(&user).Updates(updates)
	c.JSON(200, gin.H{"data": user})
}

func adminDeleteUser(c *gin.Context) {
	id := c.Param("id")
	db.DB.Delete(&db.User{}, id)
	c.JSON(200, gin.H{"message": "deleted"})
}

func adminListKeys(c *gin.Context) {
	var keys []db.APIKey
	db.DB.Preload("User").Order("id desc").Find(&keys)
	c.JSON(200, gin.H{"data": keys})
}

func adminCreateKey(c *gin.Context) {
	var req struct {
		UserID uint   `json:"user_id" binding:"required"`
		Name   string `json:"name"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	enabled := true
	key := GenerateAPIKey()
	apiKey := db.APIKey{
		Key:       key,
		UserID:    req.UserID,
		Name:      req.Name,
		IsEnabled: &enabled,
	}

	db.DB.Create(&apiKey)
	c.JSON(201, gin.H{"data": apiKey, "key": key})
}

func adminDeleteKey(c *gin.Context) {
	id := c.Param("id")
	db.DB.Delete(&db.APIKey{}, id)
	c.JSON(200, gin.H{"message": "deleted"})
}

func adminListProviders(c *gin.Context) {
	var providers []db.ProviderConfig
	db.DB.Order("priority desc, id asc").Find(&providers)
	c.JSON(200, gin.H{"data": providers})
}

func adminCreateProvider(c *gin.Context) {
	var req struct {
		Name     string `json:"name" binding:"required"`
		Type     string `json:"type" binding:"required"`
		Category string `json:"category"`
		BaseURL  string `json:"base_url" binding:"required"`
		APIKey   string `json:"api_key" binding:"required"`
		Timeout  int    `json:"timeout"`
		Priority int    `json:"priority"`
		Weight   int    `json:"weight"`
		Remark   string `json:"remark"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	enabled := true
	category := req.Category
	if category == "" {
		category = db.CategoryAPI
	}

	provider := db.ProviderConfig{
		Name:      req.Name,
		Type:      req.Type,
		Category:  category,
		BaseURL:   req.BaseURL,
		APIKey:    req.APIKey,
		IsEnabled: &enabled,
		Timeout:   req.Timeout,
		Priority:  req.Priority,
		Weight:    req.Weight,
		Remark:    req.Remark,
	}

	if provider.Timeout == 0 {
		provider.Timeout = 60
	}
	if provider.Weight == 0 {
		provider.Weight = 1
	}

	db.DB.Create(&provider)
	c.JSON(201, gin.H{"data": provider})
}

func adminUpdateProvider(c *gin.Context) {
	id := c.Param("id")
	var provider db.ProviderConfig
	if err := db.DB.First(&provider, id).Error; err != nil {
		c.JSON(404, gin.H{"error": "provider not found"})
		return
	}

	var req struct {
		Name      string `json:"name"`
		Type      string `json:"type"`
		Category  string `json:"category"`
		BaseURL   string `json:"base_url"`
		APIKey    string `json:"api_key"`
		IsEnabled *bool  `json:"is_enabled"`
		Timeout   int    `json:"timeout"`
		Priority  *int   `json:"priority"`
		Weight    *int   `json:"weight"`
		Remark    string `json:"remark"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]interface{}{}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Type != "" {
		updates["type"] = req.Type
	}
	if req.Category != "" {
		updates["category"] = req.Category
	}
	if req.BaseURL != "" {
		updates["base_url"] = req.BaseURL
	}
	if req.APIKey != "" {
		updates["api_key"] = req.APIKey
	}
	if req.IsEnabled != nil {
		updates["is_enabled"] = *req.IsEnabled
	}
	if req.Timeout > 0 {
		updates["timeout"] = req.Timeout
	}
	if req.Priority != nil {
		updates["priority"] = *req.Priority
	}
	if req.Weight != nil {
		updates["weight"] = *req.Weight
	}
	if req.Remark != "" {
		updates["remark"] = req.Remark
	}

	db.DB.Model(&provider).Updates(updates)
	c.JSON(200, gin.H{"data": provider})
}

func adminDeleteProvider(c *gin.Context) {
	id := c.Param("id")
	db.DB.Delete(&db.ProviderConfig{}, id)
	c.JSON(200, gin.H{"message": "deleted"})
}

func adminListProviderPresets(c *gin.Context) {
	presets := db.GetAllPresets()
	c.JSON(200, gin.H{"data": presets})
}

func adminListModels(c *gin.Context) {
	var models []db.ModelMapping
	db.DB.Preload("ProviderConfig").Order("id asc").Find(&models)
	c.JSON(200, gin.H{"data": models})
}

func adminCreateModel(c *gin.Context) {
	var req struct {
		Name             string `json:"name" binding:"required"`
		ProviderConfigID uint   `json:"provider_config_id" binding:"required"`
		UpstreamModel    string `json:"upstream_model"`
		Weight           int    `json:"weight"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	enabled := true
	mapping := db.ModelMapping{
		Name:             req.Name,
		ProviderConfigID: req.ProviderConfigID,
		UpstreamModel:    req.UpstreamModel,
		IsEnabled:        &enabled,
		Weight:           req.Weight,
	}
	if mapping.Weight == 0 {
		mapping.Weight = 1
	}

	db.DB.Create(&mapping)
	c.JSON(201, gin.H{"data": mapping})
}

func adminUpdateModel(c *gin.Context) {
	id := c.Param("id")
	var mapping db.ModelMapping
	if err := db.DB.First(&mapping, id).Error; err != nil {
		c.JSON(404, gin.H{"error": "model mapping not found"})
		return
	}

	var req struct {
		Name             string `json:"name"`
		ProviderConfigID uint   `json:"provider_config_id"`
		UpstreamModel    string `json:"upstream_model"`
		IsEnabled        *bool  `json:"is_enabled"`
		Weight           *int   `json:"weight"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]interface{}{}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.ProviderConfigID > 0 {
		updates["provider_config_id"] = req.ProviderConfigID
	}
	if req.UpstreamModel != "" {
		updates["upstream_model"] = req.UpstreamModel
	}
	if req.IsEnabled != nil {
		updates["is_enabled"] = *req.IsEnabled
	}
	if req.Weight != nil {
		updates["weight"] = *req.Weight
	}

	db.DB.Model(&mapping).Updates(updates)
	c.JSON(200, gin.H{"data": mapping})
}

func adminDeleteModel(c *gin.Context) {
	id := c.Param("id")
	db.DB.Delete(&db.ModelMapping{}, id)
	c.JSON(200, gin.H{"message": "deleted"})
}

func adminGetStats(c *gin.Context) {
	var totalUsers int64
	var totalKeys int64
	var totalProviders int64
	var totalModels int64

	db.DB.Model(&db.User{}).Count(&totalUsers)
	db.DB.Model(&db.APIKey{}).Count(&totalKeys)
	db.DB.Model(&db.ProviderConfig{}).Count(&totalProviders)
	db.DB.Model(&db.ModelMapping{}).Count(&totalModels)

	// Today's usage
	today := getCurrentDate()
	var todayTokens int64
	var todayCalls int64
	db.DB.Model(&db.DailyUsage{}).Where("date = ?", today).
		Select("COALESCE(SUM(token_count), 0), COALESCE(SUM(call_count), 0)").
		Row().Scan(&todayTokens, &todayCalls)

	c.JSON(200, gin.H{
		"total_users":     totalUsers,
		"total_keys":      totalKeys,
		"total_providers": totalProviders,
		"total_models":    totalModels,
		"today_tokens":    todayTokens,
		"today_calls":     todayCalls,
	})
}

func adminGetUsage(c *gin.Context) {
	userID := c.Query("user_id")
	date := c.Query("date")
	query := db.DB.Model(&db.UsageLog{})
	if userID != "" {
		query = query.Where("user_id = ?", userID)
	}
	if date != "" {
		query = query.Where("created_at >= ?", date)
	}

	var logs []db.UsageLog
	query.Order("created_at desc").Limit(100).Find(&logs)
	c.JSON(200, gin.H{"data": logs})
}

func getCurrentDate() string {
	return time.Now().Format("2006-01-02")
}

// LBConfig represents load balancing configuration for a model
func adminGetLBConfig(c *gin.Context) {
	modelName := c.Param("name")

	var mappings []db.ModelMapping
	db.DB.Preload("ProviderConfig").Where("name = ? AND is_enabled = 1", modelName).Find(&mappings)

	if len(mappings) == 0 {
		c.JSON(404, gin.H{"error": "model not found"})
		return
	}

	// Build provider list with weights
	providers := make([]gin.H, 0, len(mappings))
	for _, m := range mappings {
		providers = append(providers, gin.H{
			"mapping_id":    m.ID,
			"provider_id":   m.ProviderConfigID,
			"provider_name": m.ProviderConfig.Name,
			"provider_type": m.ProviderConfig.Type,
			"weight":        m.Weight,
			"priority":      m.ProviderConfig.Priority,
		})
	}

	strategy := "round-robin"
	if len(mappings) == 1 {
		strategy = "single"
	}

	c.JSON(200, gin.H{
		"model":      modelName,
		"strategy":   strategy,
		"providers":  providers,
		"total":      len(providers),
	})
}

func adminUpdateLBStrategy(c *gin.Context) {
	modelName := c.Param("name")

	var req struct {
		MappingWeights []struct {
			MappingID uint `json:"mapping_id"`
			Weight    int  `json:"weight"`
		} `json:"mapping_weights"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// Update weights
	for _, mw := range req.MappingWeights {
		if mw.Weight > 0 {
			db.DB.Model(&db.ModelMapping{}).Where("id = ?", mw.MappingID).Update("weight", mw.Weight)
		}
	}

	c.JSON(200, gin.H{"message": "updated", "model": modelName})
}

// adminChangeUserPassword allows admin to reset any user's password
func adminChangeUserPassword(c *gin.Context) {
	id := c.Param("id")
	var user db.User
	if err := db.DB.First(&user, id).Error; err != nil {
		c.JSON(404, gin.H{"error": "user not found"})
		return
	}

	var req struct {
		Password        string `json:"password" binding:"required"`
		PasswordConfirm string `json:"password_confirm" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	if req.Password != req.PasswordConfirm {
		c.JSON(400, gin.H{"error": "两次输入的密码不一致"})
		return
	}

	if len(req.Password) < 6 {
		c.JSON(400, gin.H{"error": "密码至少 6 个字符"})
		return
	}

	hash, err := BcryptHash(req.Password)
	if err != nil {
		c.JSON(500, gin.H{"error": "failed to hash password"})
		return
	}

	db.DB.Model(&user).Update("password_hash", hash)
	c.JSON(200, gin.H{"message": "密码已修改"})
}

// ===== Model Pricing Management =====

func adminListPrices(c *gin.Context) {
	var prices []db.ModelPrice
	db.DB.Order("model ASC").Find(&prices)
	c.JSON(200, gin.H{"data": prices})
}

func adminCreatePrice(c *gin.Context) {
	var req struct {
		Model           string  `json:"model" binding:"required"`
		InputPerMillion  float64 `json:"input_per_million"`
		OutputPerMillion float64 `json:"output_per_million"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	price := db.ModelPrice{
		Model:           req.Model,
		InputPerMillion:  req.InputPerMillion,
		OutputPerMillion: req.OutputPerMillion,
	}

	db.DB.Create(&price)
	c.JSON(201, gin.H{"data": price})
}

func adminUpdatePrice(c *gin.Context) {
	id := c.Param("id")
	var price db.ModelPrice
	if err := db.DB.First(&price, id).Error; err != nil {
		c.JSON(404, gin.H{"error": "price not found"})
		return
	}

	var req struct {
		InputPerMillion  *float64 `json:"input_per_million"`
		OutputPerMillion *float64 `json:"output_per_million"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]interface{}{}
	if req.InputPerMillion != nil {
		updates["input_per_million"] = *req.InputPerMillion
	}
	if req.OutputPerMillion != nil {
		updates["output_per_million"] = *req.OutputPerMillion
	}

	db.DB.Model(&price).Updates(updates)
	c.JSON(200, gin.H{"data": price})
}

func adminDeletePrice(c *gin.Context) {
	id := c.Param("id")
	db.DB.Delete(&db.ModelPrice{}, id)
	c.JSON(200, gin.H{"message": "deleted"})
}
