package usecase

import (
	"ddp0_grader/app/models"
	"ddp0_grader/app/repository"
)

type getProblemUseCase struct {
	problemRepo repository.ProblemRepository
}

type GetProblemInput struct {
	ID string `json:"id" binding:"required"`
}

type GetProblemUseCase interface {
	Execute(input GetProblemInput) (models.Problem, error)
}

func NewGetProblemUseCase(problemRepo repository.ProblemRepository) *getProblemUseCase {
	return &getProblemUseCase{problemRepo: problemRepo}
}

func (uc *getProblemUseCase) Execute(input GetProblemInput) (models.Problem, error) {
	problem, err := uc.problemRepo.GetProblemByIDWithPreloaded(input.ID)
	if err != nil {
		return models.Problem{}, err
	}

	return *problem, nil
}
