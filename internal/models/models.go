package models

import (
	"log"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Setup() {
	var err error
	log.Printf("connecting to db")

	// github.com/mattn/go-sqlite3
	DB, err = gorm.Open(sqlite.Open("db/gorm.db"), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to open DB: %v", err)
	}

	if err := DB.AutoMigrate(&Tag{}, &Task{}); err != nil {
		log.Fatalf("failed to migrate DB: %v", err)
	}
}
