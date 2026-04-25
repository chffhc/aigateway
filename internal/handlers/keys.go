package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vicecatcher/aigateway/internal/db"
	"github.com/vicecatcher/aigateway/internal/middleware"
	"golang.org/x/crypto/bcrypt"
)

func GenerateAPIKey() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return "sk-" + hex.EncodeToString(bytes)
}

func BcryptHash(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func RegisterKeys(g *gin.Engine) {
	api := g.Group("/api/keys")
	api.Use(middleware.APIKeyAuth())

	api.GET("", tenantListKeys)
	api.POST("", tenantCreateKey)
	api.PUT("/:id", tenantUpdateKey)
	api.DELETE("/:id", tenantDeleteKey)
}

func tenantListKeys(c *gin.Context) {
	userCtx, _ := middleware.GetUserContext(c)

	var keys []db.APIKey
	db.DB.Where("user_id = ?", userCtx.UserID).Order("id desc").Find(&keys)
	c.JSON(200, gin.H{"data": keys})
}

func tenantCreateKey(c *gin.Context) {
	userCtx, _ := middleware.GetUserContext(c)

	var req struct {
		Name string `json:"name"`
	}
	c.ShouldBindJSON(&req)

	enabled := true
	key := GenerateAPIKey()
	apiKey := db.APIKey{
		Key:       key,
		UserID:    userCtx.UserID,
		Name:      req.Name,
		IsEnabled: &enabled,
	}

	db.DB.Create(&apiKey)
	c.JSON(201, gin.H{"data": apiKey, "key": key})
}

func tenantUpdateKey(c *gin.Context) {
	userCtx, _ := middleware.GetUserContext(c)
	id := c.Param("id")

	var apiKey db.APIKey
	if err := db.DB.Where("id = ? AND user_id = ?", id, userCtx.UserID).First(&apiKey).Error; err != nil {
		c.JSON(404, gin.H{"error": "key not found"})
		return
	}

	var req struct {
		Name      string     `json:"name"`
		IsEnabled *bool      `json:"is_enabled"`
		ExpiredAt *time.Time `json:"expired_at"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]interface{}{}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.IsEnabled != nil {
		updates["is_enabled"] = *req.IsEnabled
	}
	if req.ExpiredAt != nil {
		updates["expired_at"] = *req.ExpiredAt
	}

	db.DB.Model(&apiKey).Updates(updates)
	c.JSON(200, gin.H{"data": apiKey})
}

func tenantDeleteKey(c *gin.Context) {
	userCtx, _ := middleware.GetUserContext(c)
	id := c.Param("id")

	result := db.DB.Where("id = ? AND user_id = ?", id, userCtx.UserID).Delete(&db.APIKey{})
	if result.RowsAffected == 0 {
		c.JSON(404, gin.H{"error": "key not found"})
		return
	}

	c.JSON(200, gin.H{"message": "key revoked"})
}

func RegisterAuth(g *gin.Engine) {
	g.POST("/api/auth/login", loginHandler)
	g.POST("/api/auth/register", registerHandler)
}

func loginHandler(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	var user db.User
	if err := db.DB.Where("username = ?", req.Username).First(&user).Error; err != nil {
		c.JSON(401, gin.H{"error": "invalid credentials"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		c.JSON(401, gin.H{"error": "invalid credentials"})
		return
	}

	if user.IsEnabled != nil && !*user.IsEnabled {
		c.JSON(403, gin.H{"error": "account disabled"})
		return
	}

	c.JSON(200, gin.H{
		"data": gin.H{
			"user_id":    user.ID,
			"username":   user.Username,
			"role":       user.Role,
			"quota_type": user.QuotaType,
			"token_quota": user.TokenQuota,
			"token_used":  user.TokenUsed,
			"call_quota":  user.CallQuota,
			"call_used":   user.CallUsed,
		},
	})
}

func registerHandler(c *gin.Context) {
	// Registration can be disabled or require admin approval
	// For now, allow self-registration as tenant
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
		Email    string `json:"email"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// Check if username exists
	var count int64
	db.DB.Model(&db.User{}).Where("username = ?", req.Username).Count(&count)
	if count > 0 {
		c.JSON(409, gin.H{"error": "username already exists"})
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
		Role:       "tenant",
		QuotaType:  "token",
		Email:      req.Email,
		IsEnabled:  &enabled,
	}

	db.DB.Create(&user)
	c.JSON(201, gin.H{"data": gin.H{"user_id": user.ID, "username": user.Username}})
}

func RegisterDashboard(g *gin.Engine) {
	api := g.Group("/api/dashboard")
	api.Use(middleware.APIKeyAuth())

	api.GET("/me", dashboardMe)
	api.GET("/usage", dashboardUsage)
	api.PUT("/password", dashboardChangePassword)
}

func dashboardMe(c *gin.Context) {
	userCtx, _ := middleware.GetUserContext(c)

	var user db.User
	if err := db.DB.First(&user, userCtx.UserID).Error; err != nil {
		c.JSON(404, gin.H{"error": "user not found"})
		return
	}

	// Count API keys
	var keyCount int64
	db.DB.Model(&db.APIKey{}).Where("user_id = ?", userCtx.UserID).Count(&keyCount)

	c.JSON(200, gin.H{
		"data": gin.H{
			"user_id":     user.ID,
			"username":    user.Username,
			"role":        user.Role,
			"quota_type":  user.QuotaType,
			"token_quota": user.TokenQuota,
			"token_used":  user.TokenUsed,
			"call_quota":  user.CallQuota,
			"call_used":   user.CallUsed,
			"key_count":   keyCount,
			"created_at":  user.CreatedAt,
		},
	})
}

func dashboardUsage(c *gin.Context) {
	userCtx, _ := middleware.GetUserContext(c)

	var logs []db.UsageLog
	db.DB.Where("user_id = ?", userCtx.UserID).
		Order("created_at desc").
		Limit(100).
		Find(&logs)

	// Daily summary
	today := time.Now().Format("2006-01-02")
	var daily db.DailyUsage
	db.DB.Where("user_id = ? AND date = ?", userCtx.UserID, today).First(&daily)

	c.JSON(200, gin.H{
		"data": gin.H{
			"today_calls":  daily.CallCount,
			"today_tokens": daily.TokenCount,
			"recent_logs":  logs,
		},
	})
}

func dashboardChangePassword(c *gin.Context) {
	userCtx, _ := middleware.GetUserContext(c)

	var user db.User
	if err := db.DB.First(&user, userCtx.UserID).Error; err != nil {
		c.JSON(404, gin.H{"error": "user not found"})
		return
	}

	var req struct {
		OldPassword     string `json:"old_password" binding:"required"`
		Password        string `json:"password" binding:"required"`
		PasswordConfirm string `json:"password_confirm" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// Verify old password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.OldPassword)); err != nil {
		c.JSON(400, gin.H{"error": "旧密码不正确"})
		return
	}

	// Check new passwords match
	if req.Password != req.PasswordConfirm {
		c.JSON(400, gin.H{"error": "两次输入的新密码不一致"})
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
