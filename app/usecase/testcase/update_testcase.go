package usecase

import (
	"ddp0_grader/app/repository"
)

type UpdateTestCaseInput struct {
	ID        string `json:"id" binding:"required"`
	ProblemID string `json:"problem_id" binding:"required"`
	Input     string `json:"input" binding:"required"`
	Output    string `json:"output" binding:"required"`
	IsHidden  bool   `json:"is_hidden"`
}

type UpdateTestCaseUseCase interface {
	Execute(input UpdateTestCaseInput) error
}

type updateTestCaseUseCase struct {
	testCaseRepo repository.TestCaseRepository
}

func NewUpdateTestCaseUseCase(testCaseRepo repository.TestCaseRepository) UpdateTestCaseUseCase {
	return &updateTestCaseUseCase{testCaseRepo}
}

func (uc *updateTestCaseUseCase) Execute(input UpdateTestCaseInput) error {
	testCase, err := uc.testCaseRepo.GetTestCaseByID(input.ID)
	if err != nil {
		return err
	}

	testCase.ProblemID = input.ProblemID
	testCase.Input = input.Input
	testCase.Output = input.Output
	testCase.IsHidden = input.IsHidden

	if err := uc.testCaseRepo.SaveTestCase(testCase); err != nil {
		return err
	}

	return nil
}
