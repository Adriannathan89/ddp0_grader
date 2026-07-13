package models

import (
	"ddp0_grader/app/config"
	"log"
)

type Submission struct {
	ID              string           `gorm:"primaryKey" json:"id"`
	ProblemID       string           `gorm:"not null" json:"problem_id"`
	UserID          string           `gorm:"not null" json:"user_id"`
	SourceCode      string           `gorm:"not null" json:"source_code"`
	Status          string           `gorm:"not null" json:"status"`
	Score           int              `gorm:"not null" json:"score"`
	RunTime         int              `gorm:"not null" json:"run_time"`
	ErrorMessage    *string          `gorm:"default:null" json:"error_message"`
	CreatedAt       string           `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt       string           `gorm:"autoUpdateTime" json:"updated_at"`
	TestCaseResults []TestCaseResult `gorm:"foreignKey:SubmissionID" json:"test_case_results"`
}

func (s *Submission) Save() error {
	if err := config.DB.Save(s).Error; err != nil {
		log.Printf("Error saving submission: %v", err)
		return err
	}
	return nil
}

func (s *Submission) Delete() error {
	if err := config.DB.Delete(s).Error; err != nil {
		log.Printf("Error deleting submission: %v", err)
		return err
	}
	return nil
}

func GetSubmissionByID(id string) (*Submission, error) {
	var submission Submission
	if err := config.DB.First(&submission, "id = ?", id).Error; err != nil {
		log.Printf("Error retrieving submission by ID: %v", err)
		return nil, err
	}
	return &submission, nil
}
