package models

import "time"

type TestCase struct {
	ID        string    `gorm:"primaryKey" json:"id"`
	ProblemID string    `gorm:"not null" json:"problem_id"`
	Input     string    `gorm:"not null" json:"input"`
	Output    string    `gorm:"not null" json:"output"`
	IsHidden  bool      `gorm:"default:true" json:"is_hidden"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
