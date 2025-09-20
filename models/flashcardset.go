package models

import (
	"time"

	"gorm.io/gorm"
)

// FlashcardSet represents a collection of flashcards
type FlashcardSet struct {
	gorm.Model
	Title    string    `gorm:"not null;size:100"`
	UserID   uint      `gorm:"not null"`
	PublicID string    `gorm:"size:100;uniqueIndex"`
	User     User      `gorm:"foreignKey:UserID" json:"-"`
	MindMaps []MindMap `gorm:"foreignKey:SetID"` // Associated mind maps

	Flashcards []Flashcard `gorm:"foreignKey:SetID"`

	IsPublic    bool       `gorm:"default:false"`
	LastStudied *time.Time `gorm:"default:null"`
}
