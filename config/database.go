package config

import (
	"github.com/andrewpaige1/nodebook-api/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var Database *gorm.DB

func Connect() error {
	var err error
	Database, err = gorm.Open(sqlite.Open("test.db"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	err = Database.AutoMigrate(&models.Flashcard{}, &models.User{}, &models.FlashcardSet{})
	if err != nil {
		panic("failed to auto migrate database")
	}

	return nil
}
