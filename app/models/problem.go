package models

import "time"

const (
	TagMath        = "math"
	TagVariable    = "variable"
	TagOperational = "operational"
	TagConditional = "conditional"
	TagLoop        = "loop"
	TagFunction    = "function"
)

const (
	DifficultyEasy   = "easy"
	DifficultyMedium = "medium"
	DifficultyHard   = "hard"
)

type Problem struct {
	ID          string     `gorm:"primaryKey" json:"id"`
	Title       string     `gorm:"not null" json:"title"`
	Description string     `gorm:"not null" json:"description"`
	Author      string     `gorm:"not null" json:"created_by"`
	Tag         string     `gorm:"not null" json:"tag"`
	Difficulty  string     `gorm:"not null" json:"difficulty"`
	TimeLimit   int        `gorm:"not null" json:"time_limit"`
	MemoryLimit int        `gorm:"not null" json:"memory_limit"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	TestCases   []TestCase `gorm:"foreignKey:ProblemID; constraint:OnUpdate:CASCADE,OnDelete:SET NULL" json:"test_cases"`
}
