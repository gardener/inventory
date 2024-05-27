package models

import (
	"time"
)

// Model is the base model in the inventory system.
type Model struct {
	ID        uint64    `bun:"id,pk,autoincrement"`
	CreatedAt time.Time `bun:"created_at,notnull,default:current_timestamp"`
	UpdatedAt time.Time `bun:"updated_at,notnull,default:current_timestamp"`
}
