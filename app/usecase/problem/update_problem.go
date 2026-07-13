package usecase

import (
	"ddp0_grader/app/repository"
)

type UpdateProblemInput struct {
	ID          string `json:"id" binding:"required"`
	Title       string `json:"title"`
	Description string `json:"description"`
	TimeLimit   int    `json:"time_limit"`
	MemoryLimit int    `json:"memory_limit"`
}

type UpdateProblemUseCase interface {
	Execute(input UpdateProblemInput) error
}

type updateProblemUseCase struct {
	problemRepo repository.ProblemRepository
}

func NewUpdateProblemUseCase(problemRepo repository.ProblemRepository) UpdateProblemUseCase {
	return &updateProblemUseCase{problemRepo}
}

func (uc *updateProblemUseCase) Execute(input UpdateProblemInput) error {
	problem, err := uc.problemRepo.GetProblemByID(input.ID)
	if err != nil {
		return err
	}

	if input.Title != "" {
		problem.Title = input.Title
	}
	if input.Description != "" {
		problem.Description = input.Description
	}
	if input.TimeLimit > 0 {
		problem.TimeLimit = input.TimeLimit
	}
	if input.MemoryLimit > 0 {
		problem.MemoryLimit = input.MemoryLimit
	}

	return uc.problemRepo.SaveProblem(problem)
}
