package repository

import (
	"context"
	"ddp0_grader/app/models"
	"log"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type TestCaseRepository interface {
	GetTestCaseByID(id string) (*models.TestCase, error)
	GetTestCasesByProblemID(problemID string) ([]models.TestCase, error)
	CreateTestCaseIfUnderLimit(ctx context.Context, testCase *models.TestCase, limit int) (bool, error)
	SaveTestCase(testCase *models.TestCase) error
	DeleteTestCase(testCase *models.TestCase) error
}

// CreateTestCaseIfUnderLimit serializes testcase creation per problem. Locking
// the parent problem row prevents concurrent requests from both passing the
// count check and exceeding the configured limit.
func (r *testcaseRepository) CreateTestCaseIfUnderLimit(ctx context.Context, testCase *models.TestCase, limit int) (created bool, err error) {
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var problem models.Problem
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Select("id").First(&problem, "id = ?", testCase.ProblemID).Error; err != nil {
			return err
		}

		var count int64
		if err := tx.Model(&models.TestCase{}).Where("problem_id = ?", testCase.ProblemID).Count(&count).Error; err != nil {
			return err
		}
		if count >= int64(limit) {
			return nil
		}
		if err := tx.Create(testCase).Error; err != nil {
			return err
		}
		created = true
		return nil
	})
	return created, err
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
	// Keep the order deterministic for both the participant and admin views:
	// public sample cases come first, then cases retain their creation order.
	if err := r.db.Where("problem_id = ?", problemID).Order("is_hidden ASC").Order("created_at ASC").Find(&testCases).Error; err != nil {
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
