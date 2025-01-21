package config

import (
	"github.com/andrewpaige1/nodebook-api/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var Database *gorm.DB

func Connect() error {
	var err error

	// Open SQLite database - this will create a file named nodebook.db
	Database, err = gorm.Open(sqlite.Open("test.db"), &gorm.Config{})
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
