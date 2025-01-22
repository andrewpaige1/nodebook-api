package models

import (
	"gorm.io/gorm"
)

type MindMapNodeLayout struct {
	gorm.Model
	MindMapID   uint    `gorm:"not null"`
	FlashcardID uint    `gorm:"not null"`
	XPosition   float64 `gorm:"not null"`
	YPosition   float64 `gorm:"not null"`
	Data        string  `gorm:"not null;size:200"`
}
