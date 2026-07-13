package usecase

import (
	"ddp0_grader/app/models"
	"ddp0_grader/app/repository"

	"github.com/google/uuid"
)

type CreateSubmissionInput struct {
	ProblemID  string `json:"problem_id" binding:"required"`
	UserID     string `json:"user_id" binding:"required"`
	SourceCode string `json:"source_code" binding:"required"`
	Score      int    `json:"score" binding:"required"`
}

type CreateSubmissionUseCase interface {
	Execute(input CreateSubmissionInput) (models.Submission, error)
}

type createSubmissionUseCase struct {
	submissionRepo repository.SubmissionRepository
}

func NewCreateSubmissionUseCase(submissionRepo repository.SubmissionRepository) CreateSubmissionUseCase {
	return &createSubmissionUseCase{submissionRepo}
}

func (uc *createSubmissionUseCase) Execute(input CreateSubmissionInput) (models.Submission, error) {
	submission := &models.Submission{
		ID:         uuid.New().String(),
		ProblemID:  input.ProblemID,
		UserID:     input.UserID,
		SourceCode: input.SourceCode,
		Score:      input.Score,
		Status:     "queued",
	}

	if err := uc.submissionRepo.SaveSubmission(submission); err != nil {
		return models.Submission{}, err
	}

	return *submission, nil
}
