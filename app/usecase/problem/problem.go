package problem

import (
	"context"
	"errors"
	"strings"

	"ddp0_grader/app/models"
	"ddp0_grader/app/repository"

	"github.com/google/uuid"
)

var ErrInvalidInput = errors.New("invalid problem input")

type CreateInput struct {
	Title       string
	Description string
	Author      string
	Tag         string
	Difficulty  string
	TimeLimit   int
	MemoryLimit int
}

type UpdateInput = CreateInput

type UseCase interface {
	Create(ctx context.Context, input CreateInput) (models.Problem, error)
	GetAll(ctx context.Context) ([]models.Problem, error)
	GetByID(ctx context.Context, id string) (models.Problem, error)
	Update(ctx context.Context, id string, input UpdateInput) (models.Problem, error)
	Delete(ctx context.Context, id string) error
}

type useCase struct {
	repo repository.ProblemRepository
}

func NewUseCase(repo repository.ProblemRepository) UseCase {
	return &useCase{repo: repo}
}

func (uc *useCase) Create(_ context.Context, input CreateInput) (models.Problem, error) {
	problem := models.Problem{ID: uuid.NewString()}
	if err := applyInput(&problem, input); err != nil {
		return models.Problem{}, err
	}
	if err := uc.repo.SaveProblem(&problem); err != nil {
		return models.Problem{}, err
	}
	return problem, nil
}

func (uc *useCase) GetAll(_ context.Context) ([]models.Problem, error) {
	return uc.repo.GetAllProblems()
}

func (uc *useCase) GetByID(_ context.Context, id string) (models.Problem, error) {
	problem, err := uc.repo.GetProblemByID(strings.TrimSpace(id))
	if err != nil {
		return models.Problem{}, err
	}
	return *problem, nil
}

func (uc *useCase) Update(_ context.Context, id string, input UpdateInput) (models.Problem, error) {
	problem, err := uc.repo.GetProblemByID(strings.TrimSpace(id))
	if err != nil {
		return models.Problem{}, err
	}
	if err := applyInput(problem, input); err != nil {
		return models.Problem{}, err
	}
	if err := uc.repo.SaveProblem(problem); err != nil {
		return models.Problem{}, err
	}
	return *problem, nil
}

func (uc *useCase) Delete(_ context.Context, id string) error {
	problem, err := uc.repo.GetProblemByID(strings.TrimSpace(id))
	if err != nil {
		return err
	}
	return uc.repo.DeleteProblem(problem)
}

func applyInput(problem *models.Problem, input CreateInput) error {
	problem.Title = strings.TrimSpace(input.Title)
	problem.Description = strings.TrimSpace(input.Description)
	problem.Author = strings.TrimSpace(input.Author)
	problem.Tag = strings.ToLower(strings.TrimSpace(input.Tag))
	problem.Difficulty = strings.ToLower(strings.TrimSpace(input.Difficulty))
	problem.TimeLimit = input.TimeLimit
	problem.MemoryLimit = input.MemoryLimit
	if problem.Title == "" || problem.Description == "" || problem.Author == "" || !isValidTag(problem.Tag) || !isValidDifficulty(problem.Difficulty) || problem.TimeLimit <= 0 || problem.MemoryLimit <= 0 {
		return ErrInvalidInput
	}
	return nil
}

func isValidTag(tag string) bool {
	switch tag {
	case models.TagMath, models.TagVariable, models.TagOperational, models.TagConditional, models.TagLoop, models.TagFunction:
		return true
	default:
		return false
	}
}

func isValidDifficulty(difficulty string) bool {
	switch difficulty {
	case models.DifficultyEasy, models.DifficultyMedium, models.DifficultyHard:
		return true
	default:
		return false
	}
}
