package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var Database *gorm.DB

func Connect() error {
	var err error
	var dialect gorm.Dialector
	err = godotenv.Load()
	if err != nil {
		log.Printf("Warning: .env file not found, environment variables might not be loaded: %v", err)
	}
	// Check if DB_URL is set
	dbURL := os.Getenv("DB_URL")
	if dbURL != "" {
		dialect = postgres.Open(dbURL)
	}

	// Open database connection
	Database, err = gorm.Open(dialect, &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	// Auto migrate the schema
	/*err = Database.AutoMigrate(&models.Flashcard{}, &models.User{}, &models.FlashcardSet{}, &models.MindMap{},
		&models.MindMapConnection{},
		&models.MindMapNodeLayout{},
		&models.BlocksScore{},
	)
	if err != nil {
		panic("failed to auto migrate database")
	}*/

	return nil
}
