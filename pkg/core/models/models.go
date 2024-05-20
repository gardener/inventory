package models

import (
	"time"
)

// Base is the base model in the inventory system.
//
// The model is similar to the base gorm.Model, except that it doesn't include
// the field for soft-deletes.
type Base struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time
}
