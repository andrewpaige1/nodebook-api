package models

import "gorm.io/gorm"

type MindMapConnection struct {
	gorm.Model
	MindMapID    uint   `gorm:"not null"`
	SourceID     uint   `gorm:"not null"` // References Flashcard
	TargetID     uint   `gorm:"not null"` // References Flashcard
	Relationship string `gorm:"size:200"` // Describes how the cards are related

	// References
	MindMap MindMap   `gorm:"foreignKey:MindMapID" json:"-"`
	Source  Flashcard `gorm:"foreignKey:SourceID" json:"-"`
	Target  Flashcard `gorm:"foreignKey:TargetID" json:"-"`
}
