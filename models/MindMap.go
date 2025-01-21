package models

import "gorm.io/gorm"

// MindMap represents a single mind map for a flashcard set
type MindMap struct {
	gorm.Model
	Title    string `gorm:"not null;size:100"`
	SetID    uint   `gorm:"not null"` // References FlashcardSet
	UserID   uint   `gorm:"not null"` // References User
	IsPublic bool   `gorm:"default:false"`

	// Relationships between flashcards
	Connections []MindMapConnection `gorm:"foreignKey:MindMapID"`
}
