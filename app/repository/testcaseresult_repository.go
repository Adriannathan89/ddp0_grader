package repository

import (
	"ddp0_grader/app/models"
	"log"

	"gorm.io/gorm"
)

type TestCaseResultRepository interface {
	GetTestCaseResultByID(id string) (*models.TestCaseResult, error)
	GetTestCaseResultsBySubmissionID(submissionID string) ([]models.TestCaseResult, error)
	GetTestCasesByUserIDAndSubmissionID(userID string, submissionID string) ([]models.TestCaseResult, error)
	BatchSaveTestCaseResults(testCaseResults []models.TestCaseResult) error
	DeleteTestCaseResult(testCaseResult *models.TestCaseResult) error
}

type testCaseResultRepository struct {
	db *gorm.DB
}

func NewTestCaseResultRepository(db *gorm.DB) TestCaseResultRepository {
	return &testCaseResultRepository{db}
}

func (r *testCaseResultRepository) GetTestCaseResultByID(id string) (*models.TestCaseResult, error) {
	var testCaseResult models.TestCaseResult
	if err := r.db.First(&testCaseResult, "id = ?", id).Error; err != nil {
		log.Printf("Error retrieving test case result by ID: %v", err)
		return nil, err
	}
	return &testCaseResult, nil
}

func (r *testCaseResultRepository) GetTestCaseResultsBySubmissionID(submissionID string) ([]models.TestCaseResult, error) {
	var testCaseResults []models.TestCaseResult
	if err := r.db.Where("submission_id = ?", submissionID).Find(&testCaseResults).Error; err != nil {
		log.Printf("Error retrieving test case results by submission ID: %v", err)
		return nil, err
	}
	return testCaseResults, nil
}

func (r *testCaseResultRepository) BatchSaveTestCaseResults(testCaseResults []models.TestCaseResult) error {
	if err := r.db.Create(&testCaseResults).Error; err != nil {
		log.Printf("Error batch saving test case results: %v", err)
		return err
	}
	return nil
}

func (r *testCaseResultRepository) DeleteTestCaseResult(testCaseResult *models.TestCaseResult) error {
	if err := r.db.Delete(testCaseResult).Error; err != nil {
		log.Printf("Error deleting test case result: %v", err)
		return err
	}
	return nil
}

func (r *testCaseResultRepository) GetTestCasesByUserIDAndSubmissionID(userID string, submissionID string) ([]models.TestCaseResult, error) {
	var testCaseResults []models.TestCaseResult
	if err := r.db.Joins("JOIN submissions ON submissions.id = test_case_results.submission_id").Where("submissions.user_id = ? AND test_case_results.submission_id = ?", userID, submissionID).Find(&testCaseResults).Error; err != nil {
		log.Printf("Error retrieving test case results by user ID and submission ID: %v", err)
		return nil, err
	}
	return testCaseResults, nil
}
