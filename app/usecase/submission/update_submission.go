package usecase

import (
	"ddp0_grader/app/models"
	"ddp0_grader/app/repository"
)

type UpdateSubmissionInput struct {
	ID           string  `json:"id" binding:"required"`
	ProblemID    string  `json:"problem_id"`
	UserID       string  `json:"user_id"`
	SourceCode   string  `json:"source_code"`
	Score        int     `json:"score"`
	Status       string  `json:"status"`
	RunTime      int     `json:"run_time"`
	ErrorMessage *string `json:"error_message"`
}

type UpdateSubmissionUseCase interface {
	Execute(input UpdateSubmissionInput) (models.Submission, error)
}

type updateSubmissionUseCase struct {
	submissionRepo repository.SubmissionRepository
}

func NewUpdateSubmissionUseCase(submissionRepo repository.SubmissionRepository) UpdateSubmissionUseCase {
	return &updateSubmissionUseCase{submissionRepo}
}

func (uc *updateSubmissionUseCase) Execute(input UpdateSubmissionInput) (models.Submission, error) {
	submission, err := uc.submissionRepo.GetSubmissionByID(input.ID)
	if err != nil {
		return models.Submission{}, err
	}

	if input.ProblemID != "" {
		submission.ProblemID = input.ProblemID
	}
	if input.UserID != "" {
		submission.UserID = input.UserID
	}
	if input.SourceCode != "" {
		submission.SourceCode = input.SourceCode
	}
	if input.Score != 0 {
		submission.Score = input.Score
	}
	if input.Status != "" {
		submission.Status = input.Status
	}
	if input.RunTime != 0 {
		submission.RunTime = input.RunTime
	}
	if input.ErrorMessage != nil {
		submission.ErrorMessage = input.ErrorMessage
	}

	if err := uc.submissionRepo.SaveSubmission(submission); err != nil {
		return models.Submission{}, err
	}

	return *submission, nil
}
