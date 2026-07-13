package usecase

import (
	"ddp0_grader/app/models"
	"ddp0_grader/app/repository"

	"github.com/google/uuid"
)

type CreateProblemInput struct {
	Title       string `json:"title" binding:"required"`
	Description string `json:"description" binding:"required"`
	TimeLimit   int    `json:"time_limit" binding:"required"`
	MemoryLimit int    `json:"memory_limit" binding:"required"`
}

type CreateProblemUseCase interface {
	Execute(input CreateProblemInput) (models.Problem, error)
}

type createProblemUseCase struct {
	problemRepo repository.ProblemRepository
}

func NewCreateProblemUseCase(problemRepo repository.ProblemRepository) CreateProblemUseCase {
	return &createProblemUseCase{problemRepo}
}

func (uc *createProblemUseCase) Execute(input CreateProblemInput) (models.Problem, error) {
	problem := &models.Problem{
		ID:          uuid.New().String(),
		Title:       input.Title,
		Description: input.Description,
		TimeLimit:   input.TimeLimit,
		MemoryLimit: input.MemoryLimit,
	}

	if err := uc.problemRepo.SaveProblem(problem); err != nil {
		return models.Problem{}, err
	}

	return *problem, nil
}
