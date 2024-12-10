package models

import (
	"time"

	"gorm.io/gorm"
)

// Flashcard represents an individual flashcard
type Flashcard struct {
	gorm.Model
	Term     string `gorm:"not null;size:200"`
	Solution string `gorm:"not null;size:1000"`
	Concept  string `gorm:"size:100"`

	SetID        uint         `gorm:"not null"`
	FlashcardSet FlashcardSet `gorm:"foreignKey:SetID" json:"-"`

	// Optional tracking fields
	Difficulty    int        `gorm:"default:0"`
	TimesReviewed int        `gorm:"default:0"`
	LastReviewed  *time.Time `gorm:"default:null"`
	Mastered      bool       `gorm:"default:false"`
}
