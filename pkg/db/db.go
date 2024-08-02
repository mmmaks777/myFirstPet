package db

import (
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// var DB *gorm.DB

func Connect() *gorm.DB {
	dsn := "user=Maksimka dbname=pet port=5432 sslmode=disable"
	var err error
	DB, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("failed to connect database", err)
	}

	// DB.AutoMigrate(&t.Credentials{})
	return DB
}
