package repository

import (
	"ddp0_grader/app/config"
	"ddp0_grader/app/models"
	"log"

	"gorm.io/gorm"
)

type TestCaseResultRepositoryInterface interface {
	GetTestCaseResultByID(id string) (*models.TestCaseResult, error)
	GetTestCaseResultsBySubmissionID(submissionID string) ([]models.TestCaseResult, error)
	BatchSaveTestCaseResults(testCaseResults []models.TestCaseResult) error
	DeleteTestCaseResult(testCaseResult *models.TestCaseResult) error
}

type TestCaseResultRepository struct {
	db *gorm.DB
}

func NewTestCaseResultRepository(db *gorm.DB) TestCaseResultRepositoryInterface {
	return &TestCaseResultRepository{db}
}

func (r *TestCaseResultRepository) GetTestCaseResultByID(id string) (*models.TestCaseResult, error) {
	var testCaseResult models.TestCaseResult
	if err := r.db.First(&testCaseResult, "id = ?", id).Error; err != nil {
		log.Printf("Error retrieving test case result by ID: %v", err)
		return nil, err
	}
	return &testCaseResult, nil
}

func (r *TestCaseResultRepository) GetTestCaseResultsBySubmissionID(submissionID string) ([]models.TestCaseResult, error) {
	var testCaseResults []models.TestCaseResult
	if err := r.db.Where("submission_id = ?", submissionID).Find(&testCaseResults).Error; err != nil {
		log.Printf("Error retrieving test case results by submission ID: %v", err)
		return nil, err
	}
	return testCaseResults, nil
}

func (r *TestCaseResultRepository) BatchSaveTestCaseResults(testCaseResults []models.TestCaseResult) error {
	if err := config.DB.Create(&testCaseResults).Error; err != nil {
		log.Printf("Error batch saving test case results: %v", err)
		return err
	}
	return nil
}

func (r *TestCaseResultRepository) DeleteTestCaseResult(testCaseResult *models.TestCaseResult) error {
	if err := r.db.Delete(testCaseResult).Error; err != nil {
		log.Printf("Error deleting test case result: %v", err)
		return err
	}
	return nil
}
