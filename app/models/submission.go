package models

import "time"

const (
	SubmissionStatusQueued             = "queued"
	SubmissionStatusWrongAnswer        = "wrong_answer"
	SubmissionStatusTimeLimitExceded   = "time_limit_exceded"
	SubmissionStatusMemoryLimitExceded = "memory_limit_exceded"
	SubmissionStatusSystemError        = "system_error"
	SubmissionStatusAccepted           = "accepted"
)

type Submission struct {
	ID              string           `gorm:"primaryKey" json:"id"`
	ProgressID      string           `gorm:"not null; foreignKey:ProgressID" json:"progress_id"`
	Status          string           `gorm:"default:'queued'" json:"status"`
	SourceCode      string           `gorm:"not null" json:"source_code"`
	Score           int              `gorm:"not null" json:"score"`
	TotalRunTime    int              `gorm:"not null" json:"total_run_time"`
	CreatedAt       time.Time        `json:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
	Progress        Progress         `gorm:"foreignKey:ProgressID;references:ID" json:"-"`
	TestCaseResults []TestCaseResult `gorm:"foreignKey:SubmissionID; constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"test_case_results"`
}
