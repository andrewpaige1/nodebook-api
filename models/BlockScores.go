package models

import (
	"time"
)

type BlocksScore struct {
	ID              uint         `gorm:"primaryKey"`
	UserID          uint         `gorm:"not null;index"`
	User            User         `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	FlashcardSetID  uint         `gorm:"not null;index"`
	FlashcardSet    FlashcardSet `gorm:"foreignKey:FlashcardSetID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	TimeSeconds     int          `gorm:"not null"`
	CorrectAttempts int          `gorm:"not null"`
	TotalAttempts   int          `gorm:"not null"`
	PlayedAt        time.Time    `gorm:"autoCreateTime"`
}
