package models

import "time"

type TestCaseResult struct {
	ID           string    `gorm:"primaryKey" json:"id"`
	SubmissionID string    `gorm:"not null; foreignKey:SubmissionID" json:"submission_id"`
	TestCaseID   string    `gorm:"not null; foreignKey:TestCaseID" json:"test_case_id"`
	IsPassed     bool      `gorm:"not null" json:"is_passed"`
	Verdict      string    `gorm:"not null" json:"verdict"`
	RunTime      int       `gorm:"not null" json:"run_time"`
	Feedback     *string   `gorm:"default:null" json:"feedback"`
	ErrorMessage *string   `gorm:"default:null" json:"error_message"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
