package config

import (
	"os"

	"github.com/andrewpaige1/nodebook-api/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var Database *gorm.DB

func Connect() error {
	var err error
	dbURL := os.Getenv("DB_URL")
	Database, err = gorm.Open(postgres.Open(dbURL), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	err = Database.AutoMigrate(&models.Flashcard{}, &models.User{}, &models.FlashcardSet{})
	if err != nil {
		panic("failed to auto migrate database")
	}

	return nil
}
