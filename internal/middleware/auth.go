package middleware

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vicecatcher/aigateway/internal/db"
)

const (
	RoleAdmin  = "admin"
	RoleTenant = "tenant"
)

// UserContext holds authenticated user info in gin context
type UserContext struct {
	UserID   uint
	Role     string
	APIKeyID uint
}

// APIKeyAuth authenticates requests via Bearer token
func APIKeyAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(401, gin.H{"error": "missing authorization header"})
			c.Abort()
			return
		}

		apiKey := strings.TrimPrefix(authHeader, "Bearer ")
		if apiKey == authHeader || apiKey == "" {
			c.JSON(401, gin.H{"error": "invalid authorization header"})
			c.Abort()
			return
		}

		var key db.APIKey
		if err := db.DB.Preload("User").Where("key = ? AND is_enabled = 1", apiKey).First(&key).Error; err != nil {
			c.JSON(401, gin.H{"error": "invalid api key"})
			c.Abort()
			return
		}

		// Check expiration
		if key.ExpiredAt != nil && time.Now().After(*key.ExpiredAt) {
			c.JSON(401, gin.H{"error": "api key expired"})
			c.Abort()
			return
		}

		// Check user enabled
		if key.User.IsEnabled != nil && !*key.User.IsEnabled {
			c.JSON(403, gin.H{"error": "user disabled"})
			c.Abort()
			return
		}

		// Update last used
		now := time.Now()
		key.LastUsedAt = &now
		db.DB.Model(&key).Updates(map[string]interface{}{
			"last_used_at": now,
		})

		c.Set("user_ctx", UserContext{
			UserID:   key.UserID,
			Role:     key.User.Role,
			APIKeyID: key.ID,
		})
		c.Next()
	}
}

// SessionAuth for web dashboard (cookie-based)
func SessionAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID, err := c.Cookie("session_id")
		if err != nil || sessionID == "" {
			c.Redirect(302, "/login")
			c.Abort()
			return
		}

		// Look up session (simplified: store session in memory or DB)
		var user db.User
		if err := db.DB.Where("id = ?", sessionID).First(&user).Error; err != nil {
			c.Redirect(302, "/login")
			c.Abort()
			return
		}

		if user.IsEnabled != nil && !*user.IsEnabled {
			c.HTML(403, "error.tmpl", gin.H{"message": "Your account is disabled"})
			c.Abort()
			return
		}

		c.Set("user_ctx", UserContext{
			UserID: user.ID,
			Role:   user.Role,
		})
		c.Set("user", user)
		c.Next()
	}
}

// RequireRole checks if user has required role
func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userCtx, exists := c.Get("user_ctx")
		if !exists {
			c.JSON(401, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}

		ctx := userCtx.(UserContext)
		for _, role := range roles {
			if ctx.Role == role {
				c.Next()
				return
			}
		}

		c.JSON(403, gin.H{"error": "forbidden"})
		c.Abort()
	}
}

func GetUserContext(c *gin.Context) (UserContext, bool) {
	val, exists := c.Get("user_ctx")
	if !exists {
		return UserContext{}, false
	}
	return val.(UserContext), true
}
