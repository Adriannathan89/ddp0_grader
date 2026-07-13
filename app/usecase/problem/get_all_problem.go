package usecase

import (
	"ddp0_grader/app/models"
	"ddp0_grader/app/repository"
)

type GetAllProblemsUseCase interface {
	Execute() ([]models.Problem, error)
}

type getAllProblemsUseCase struct {
	problemRepo repository.ProblemRepository
}

func NewGetAllProblemsUseCase(problemRepo repository.ProblemRepository) GetAllProblemsUseCase {
	return &getAllProblemsUseCase{problemRepo}
}

func (uc *getAllProblemsUseCase) Execute() ([]models.Problem, error) {
	problems, err := uc.problemRepo.GetAllProblems()
	if err != nil {
		return nil, err
	}

	return problems, nil
}
