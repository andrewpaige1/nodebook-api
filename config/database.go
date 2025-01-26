package config

import (
	"os"

	"github.com/andrewpaige1/nodebook-api/models"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var Database *gorm.DB

func Connect() error {
	var err error
	var dialect gorm.Dialector

	// Check if DB_URL is set (production environment)
	dbURL := os.Getenv("DB_URL")
	if dbURL != "" {
		dialect = postgres.Open(dbURL)
	} else {
		// Use SQLite for local development
		dialect = sqlite.Open("test.db")
	}

	// Open database connection
	Database, err = gorm.Open(dialect, &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	// Auto migrate the schema
	err = Database.AutoMigrate(&models.Flashcard{}, &models.User{}, &models.FlashcardSet{}, &models.MindMap{},
		&models.MindMapConnection{},
		&models.MindMapNodeLayout{},
	)
	if err != nil {
		panic("failed to auto migrate database")
	}

	return nil
}
