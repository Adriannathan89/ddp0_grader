package progress

import (
	"context"
	"errors"
	"strings"

	"ddp0_grader/app/models"
	"ddp0_grader/app/repository"
)

var ErrInvalidInput = errors.New("user_id and problem_id are required")

type UseCase interface {
	GetByUserID(ctx context.Context, userID string) ([]models.Progress, error)
	GetByUserAndProblemID(ctx context.Context, userID, problemID string) (models.Progress, error)
}

type useCase struct {
	repo repository.ProgressRepository
}

func NewUseCase(repo repository.ProgressRepository) UseCase {
	return &useCase{repo: repo}
}

func (uc *useCase) GetByUserID(_ context.Context, userID string) ([]models.Progress, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, ErrInvalidInput
	}
	return uc.repo.GetProgressesWithSubmissionsByUserID(userID)
}

func (uc *useCase) GetByUserAndProblemID(_ context.Context, userID, problemID string) (models.Progress, error) {
	userID, problemID = strings.TrimSpace(userID), strings.TrimSpace(problemID)
	if userID == "" || problemID == "" {
		return models.Progress{}, ErrInvalidInput
	}
	progress, err := uc.repo.GetProgressWithSubmissionsByProblemAndUser(problemID, userID)
	if err != nil {
		return models.Progress{}, err
	}
	return *progress, nil
}
