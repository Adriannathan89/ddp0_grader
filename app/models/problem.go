package models

import "time"

const (
	TagMath           = "math"
	TagSorting        = "sorting"
	TagDataStructure  = "Data Structure"
	TagBruteforce     = "bruteforce"
	TagImplementation = "implementation"
)

const (
	DifficultyEasy   = "easy"
	DifficultyMedium = "medium"
	DifficultyHard   = "hard"
)

type Problem struct {
	ID           string     `gorm:"primaryKey" json:"id"`
	Title        string     `gorm:"not null" json:"title"`
	Description  string     `gorm:"not null" json:"description"`
	Author       string     `gorm:"not null" json:"created_by"`
	Tag          string     `gorm:"not null" json:"tag"`
	Difficulty   string     `gorm:"not null" json:"difficulty"`
	TimeLimit    int        `gorm:"not null" json:"time_limit"`
	MemoryLimit  int        `gorm:"not null" json:"memory_limit"`
	Hint         string     `gorm:"type:text" json:"hint,omitempty"`
	IsQuiz       bool       `gorm:"not null;default:false;index:idx_problems_quiz_created_at,priority:1" json:"is_quiz"`
	ThumbnailKey string     `gorm:"column:thumbnail_key" json:"-"`
	ThumbnailURL string     `gorm:"-" json:"thumbnail_url,omitempty"`
	CreatedAt    time.Time  `gorm:"index:idx_problems_quiz_created_at,priority:2" json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	TestCases    []TestCase `gorm:"foreignKey:ProblemID; constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"test_cases"`
	Progresses   []Progress `gorm:"foreignKey:ProblemID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"-"`
}
