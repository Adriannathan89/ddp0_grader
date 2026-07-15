package repository

import (
	"ddp0_grader/app/models"
	"log"

	"gorm.io/gorm"
)

type SubmissionRepository interface {
	GetSubmissionByID(id string) (*models.Submission, error)
	GetSubmissionByIDWithPreloaded(id string) (*models.Submission, error)
	SaveSubmission(submission *models.Submission) error
	DeleteSubmission(submission *models.Submission) error
}

type submissionRepository struct {
	db *gorm.DB
}

func NewSubmissionRepository(db *gorm.DB) SubmissionRepository {
	return &submissionRepository{db}
}

func (r *submissionRepository) GetSubmissionByID(id string) (*models.Submission, error) {
	var submission models.Submission
	if err := r.db.First(&submission, "id = ?", id).Error; err != nil {
		log.Printf("Error retrieving submission by ID: %v", err)
		return nil, err
	}
	return &submission, nil
}

func (r *submissionRepository) GetSubmissionByIDWithPreloaded(id string) (*models.Submission, error) {
	var submission models.Submission
	if err := r.db.Preload("Progress").Preload("TestCaseResults").First(&submission, "id = ?", id).Error; err != nil {
		log.Printf("Error retrieving submission by ID with preloaded test case results: %v", err)
		return nil, err
	}
	return &submission, nil
}

func (r *submissionRepository) SaveSubmission(submission *models.Submission) error {
	if err := r.db.Save(submission).Error; err != nil {
		log.Printf("Error saving submission: %v", err)
		return err
	}
	return nil
}

func (r *submissionRepository) DeleteSubmission(submission *models.Submission) error {
	if err := r.db.Delete(submission).Error; err != nil {
		log.Printf("Error deleting submission: %v", err)
		return err
	}
	return nil
}
