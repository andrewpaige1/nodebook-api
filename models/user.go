package models

import "gorm.io/gorm"

// User represents a user in the system
type User struct {
	gorm.Model
	Nickname      string         `gorm:"unique;not null;size:100"`
	Auth0ID       string         `gorm:"unique;not null;size:200"`
	FlashcardSets []FlashcardSet `gorm:"foreignKey:UserID"`
	MindMaps      []MindMap
}
