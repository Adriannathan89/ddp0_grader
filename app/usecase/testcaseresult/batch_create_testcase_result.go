package usecase

import (
	"ddp0_grader/app/models"
	"ddp0_grader/app/repository"
)

type BatchCreateTestCaseResultUseCase interface {
	Execute(input BatchCreateTestCaseResultInput) error
}

type BatchCreateTestCaseResultInput struct {
	TestCaseResults []models.TestCaseResult
	// Results is kept as an alias for callers using the shorter field name.
	Results []models.TestCaseResult
}

type batchCreateTestCaseResultUseCase struct {
	resultRepo repository.TestCaseResultRepository
}

func NewBatchCreateTestCaseResultUseCase(resultRepo repository.TestCaseResultRepository) BatchCreateTestCaseResultUseCase {
	return &batchCreateTestCaseResultUseCase{resultRepo: resultRepo}
}

func (uc *batchCreateTestCaseResultUseCase) Execute(input BatchCreateTestCaseResultInput) error {
	results := input.TestCaseResults
	if len(results) == 0 {
		results = input.Results
	}
	if len(results) == 0 {
		return nil
	}
	return uc.resultRepo.BatchSaveTestCaseResults(results)
}
