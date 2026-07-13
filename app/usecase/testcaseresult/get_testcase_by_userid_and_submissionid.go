package usecase

import (
	"ddp0_grader/app/models"
	"ddp0_grader/app/repository"
)

type GetTestCaseResultsInput struct {
	UserID       string
	SubmissionID string
}

type GetTestCaseResultsUseCase interface {
	Execute(input GetTestCaseResultsInput) ([]models.TestCaseResult, error)
}

type getTestCaseResultsUseCase struct {
	resultRepo repository.TestCaseResultRepository
}

func NewGetTestCaseResultsUseCase(resultRepo repository.TestCaseResultRepository) GetTestCaseResultsUseCase {
	return &getTestCaseResultsUseCase{resultRepo: resultRepo}
}

func (uc *getTestCaseResultsUseCase) Execute(input GetTestCaseResultsInput) ([]models.TestCaseResult, error) {
	return uc.resultRepo.GetTestCasesByUserIDAndSubmissionID(input.UserID, input.SubmissionID)
}
