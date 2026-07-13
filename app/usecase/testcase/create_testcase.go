package usecase

import (
	"ddp0_grader/app/models"
	"ddp0_grader/app/repository"

	"github.com/google/uuid"
)

type CreateTestCaseInput struct {
	ProblemID string `json:"problem_id" binding:"required"`
	Input     string `json:"input" binding:"required"`
	Output    string `json:"output" binding:"required"`
	IsHidden  bool   `json:"is_hidden"`
}

type CreateTestCaseUseCase interface {
	Execute(input CreateTestCaseInput) (models.TestCase, error)
}

type createTestCaseUseCase struct {
	testCaseRepo repository.TestCaseRepository
}

func NewCreateTestCaseUseCase(testCaseRepo repository.TestCaseRepository) CreateTestCaseUseCase {
	return &createTestCaseUseCase{testCaseRepo}
}

func (uc *createTestCaseUseCase) Execute(input CreateTestCaseInput) (models.TestCase, error) {
	testCase := &models.TestCase{
		ID:        uuid.New().String(),
		ProblemID: input.ProblemID,
		Input:     input.Input,
		Output:    input.Output,
		IsHidden:  input.IsHidden,
	}

	if err := uc.testCaseRepo.SaveTestCase(testCase); err != nil {
		return models.TestCase{}, err
	}

	return *testCase, nil
}
