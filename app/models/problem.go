package models

import "time"

type Problem struct {
	ID          string     `gorm:"primaryKey" json:"id"`
	Title       string     `gorm:"not null" json:"title"`
	Description string     `gorm:"not null" json:"description"`
	TimeLimit   int        `gorm:"not null" json:"time_limit"`
	MemoryLimit int        `gorm:"not null" json:"memory_limit"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	TestCases   []TestCase `gorm:"foreignKey:ProblemID" json:"test_cases"`
}
