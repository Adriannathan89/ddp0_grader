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

const (
	maxTitleBytes       = 256
	maxDescriptionBytes = 64 << 10
	maxAuthorBytes      = 256
	maxHintBytes        = 4 << 10
	maxTimeLimitMS      = 1_000
	maxMemoryLimitMB    = 64
)

type CreateInput struct {
	Title       string
	Description string
	Author      string
	Tag         string
	Difficulty  string
	TimeLimit   int
	MemoryLimit int
	Hint        string
	IsQuiz      bool
}

type UpdateInput = CreateInput

type UseCase interface {
	Create(ctx context.Context, input CreateInput) (models.Problem, error)
	GetAll(ctx context.Context, isQuiz *bool) ([]models.Problem, error)
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

func (uc *useCase) GetAll(_ context.Context, isQuiz *bool) ([]models.Problem, error) {
	return uc.repo.GetAllProblems(isQuiz)
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

// SetThumbnailKey updates only the storage reference so normal problem edits do
// not accidentally remove an already uploaded thumbnail.
func (uc *useCase) SetThumbnailKey(_ context.Context, id, thumbnailKey string) (models.Problem, error) {
	problem, err := uc.repo.GetProblemByID(strings.TrimSpace(id))
	if err != nil {
		return models.Problem{}, err
	}
	problem.ThumbnailKey = strings.TrimSpace(thumbnailKey)
	if err := uc.repo.SaveProblem(problem); err != nil {
		return models.Problem{}, err
	}
	return *problem, nil
}

func applyInput(problem *models.Problem, input CreateInput) error {
	problem.Title = strings.TrimSpace(input.Title)
	problem.Description = strings.TrimSpace(input.Description)
	problem.Author = strings.TrimSpace(input.Author)
	problem.Tag = normalizeTag(input.Tag)
	problem.Difficulty = strings.ToLower(strings.TrimSpace(input.Difficulty))
	problem.TimeLimit = input.TimeLimit
	problem.MemoryLimit = input.MemoryLimit
	problem.Hint = strings.TrimSpace(input.Hint)
	problem.IsQuiz = input.IsQuiz
	if problem.Title == "" || problem.Description == "" || problem.Author == "" || len(problem.Title) > maxTitleBytes || len(problem.Description) > maxDescriptionBytes || len(problem.Author) > maxAuthorBytes || len(problem.Hint) > maxHintBytes || !isValidTag(problem.Tag) || !isValidDifficulty(problem.Difficulty) || problem.TimeLimit <= 0 || problem.TimeLimit > maxTimeLimitMS || problem.MemoryLimit <= 0 || problem.MemoryLimit > maxMemoryLimitMB {
		return ErrInvalidInput
	}
	return nil
}

func isValidTag(tag string) bool {
	switch tag {
	case models.TagMath, models.TagSorting, models.TagDataStructure, models.TagBruteforce, models.TagImplementation:
		return true
	default:
		return false
	}
}

func normalizeTag(value string) string {
	tag := strings.ToLower(strings.TrimSpace(value))
	switch tag {
	case "data structure", "data_structure":
		return models.TagDataStructure
	case "brute force":
		return models.TagBruteforce
	default:
		return tag
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
