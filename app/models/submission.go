package models

import "time"

type Submission struct {
	ID              string           `gorm:"primaryKey" json:"id"`
	ProblemID       string           `gorm:"not null" json:"problem_id"`
	UserID          string           `gorm:"not null" json:"user_id"`
	SourceCode      string           `gorm:"not null" json:"source_code"`
	Status          string           `gorm:"not null" json:"status"`
	Score           int              `gorm:"not null" json:"score"`
	RunTime         int              `gorm:"not null" json:"run_time"`
	ErrorMessage    *string          `gorm:"default:null" json:"error_message"`
	CreatedAt       time.Time        `json:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
	TestCaseResults []TestCaseResult `gorm:"foreignKey:SubmissionID" json:"test_case_results"`
}
