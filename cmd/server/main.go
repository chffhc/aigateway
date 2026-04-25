package main

import (
	"log"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/vicecatcher/aigateway/internal/db"
	"github.com/vicecatcher/aigateway/internal/handlers"
	"github.com/vicecatcher/aigateway/internal/proxy"
)

func main() {
	// Configuration from environment
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "aigateway.db"
	}

	// Initialize database
	if err := db.Init(dbPath); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()
	log.Println("Database initialized")

	// Seed admin user if not exists
	enabled := true
	var adminCount int64
	db.DB.Model(&db.User{}).Where("role = ?", "admin").Count(&adminCount)
	if adminCount == 0 {
		adminPassword := os.Getenv("ADMIN_PASSWORD")
		if adminPassword == "" {
			adminPassword = "admin123"
		}
		hash, _ := handlers.BcryptHash(adminPassword)
		admin := db.User{
			Username:     "admin",
			PasswordHash: hash,
			Role:         "admin",
			QuotaType:    "token",
			TokenQuota:   0, // unlimited
			IsEnabled:    &enabled,
		}
		db.DB.Create(&admin)
		log.Printf("Default admin created (username: admin, password: %s)", adminPassword)
	}

	// Seed default providers and models
	db.SeedProviders()
	db.SeedModels()
	db.SeedPrices()

	// Initialize proxy with timeout
	timeout := 60 * time.Second
	if t := os.Getenv("PROXY_TIMEOUT"); t != "" {
		if d, err := time.ParseDuration(t); err == nil {
			timeout = d
		}
	}
	p := proxy.New(timeout)

	// Setup Gin
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// CORS
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "x-api-key", "anthropic-version"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "timestamp": time.Now().Unix()})
	})

	// Register routes
	handlers.RegisterAuth(r)
	handlers.RegisterOpenAI(r, p)
	handlers.RegisterAnthropic(r, p)
	handlers.RegisterKeys(r)
	handlers.RegisterAdmin(r)
	handlers.RegisterDashboard(r)
	handlers.RegisterAPIKeyStats(r)

	// Serve static files for dashboard
	r.Static("/static", "./static")
	r.LoadHTMLGlob("templates/*")

	// Dashboard routes
	r.GET("/login", func(c *gin.Context) {
		c.HTML(200, "login.html", gin.H{})
	})

	r.GET("/admin", func(c *gin.Context) {
		c.HTML(200, "admin.html", gin.H{})
	})

	r.GET("/dashboard", func(c *gin.Context) {
		c.HTML(200, "dashboard.html", gin.H{})
	})

	log.Printf("Starting AI Gateway on :%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
