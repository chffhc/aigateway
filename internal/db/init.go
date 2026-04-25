package db

import (
	"os"
	"path/filepath"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func Init(dbPath string) error {
	if dbPath == "" {
		dbPath = "aigateway.db"
	}

	// Ensure parent directory exists
	dir := filepath.Dir(dbPath)
	if dir != "." {
		os.MkdirAll(dir, 0755)
	}

	var err error
	DB, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return err
	}

	return DB.AutoMigrate(
		&User{},
		&APIKey{},
		&ProviderConfig{},
		&ModelMapping{},
		&UsageLog{},
		&DailyUsage{},
		&ModelPrice{},
	)
}

func Close() error {
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
