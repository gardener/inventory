package models

import (
	"time"
)

// Base is the base model in the inventory system.
//
// The model is similar to the base [gorm.io/gorm.Model], except that it doesn't
// include the field for soft-deletes.
type Base struct {
	ID        uint      `gorm:"primaryKey"`
	CreatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP"`
	UpdatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP"`
}
