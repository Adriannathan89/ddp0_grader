package repository

import (
	"ddp0_grader/app/models"
	"log"

	"gorm.io/gorm"
)

type TestCaseRepository interface {
	GetTestCaseByID(id string) (*models.TestCase, error)
	GetTestCasesByProblemID(problemID string) ([]models.TestCase, error)
	SaveTestCase(testCase *models.TestCase) error
	DeleteTestCase(testCase *models.TestCase) error
}

type testcaseRepository struct {
	db *gorm.DB
}

func NewTestCaseRepository(db *gorm.DB) TestCaseRepository {
	return &testcaseRepository{db}
}

func (r *testcaseRepository) GetTestCaseByID(id string) (*models.TestCase, error) {
	var testCase models.TestCase
	if err := r.db.First(&testCase, "id = ?", id).Error; err != nil {
		log.Printf("Error retrieving test case by ID: %v", err)
		return nil, err
	}
	return &testCase, nil
}

func (r *testcaseRepository) GetTestCasesByProblemID(problemID string) ([]models.TestCase, error) {
	var testCases []models.TestCase
	if err := r.db.Where("problem_id = ?", problemID).Find(&testCases).Error; err != nil {
		log.Printf("Error retrieving test cases by problem ID: %v", err)
		return nil, err
	}
	return testCases, nil
}

func (r *testcaseRepository) SaveTestCase(testCase *models.TestCase) error {
	if err := r.db.Save(testCase).Error; err != nil {
		log.Printf("Error saving test case: %v", err)
		return err
	}
	return nil
}

func (r *testcaseRepository) DeleteTestCase(testCase *models.TestCase) error {
	if err := r.db.Delete(testCase).Error; err != nil {
		log.Printf("Error deleting test case: %v", err)
		return err
	}
	return nil
}
