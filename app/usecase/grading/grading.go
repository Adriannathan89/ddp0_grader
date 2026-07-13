package grading

import (
	"context"
	"errors"
	"time"

	"ddp0_grader/app/models"
	problemUseCase "ddp0_grader/app/usecase/problem"
	submissionUseCase "ddp0_grader/app/usecase/submission"
	resultUseCase "ddp0_grader/app/usecase/testcaseresult"
	"ddp0_grader/pkg/queue"
	"ddp0_grader/pkg/runner"

	"github.com/google/uuid"
)

type SubmitInput struct {
	ProblemID  string
	UserID     string
	SourceCode string
}

type UseCase interface {
	Submit(ctx context.Context, input SubmitInput) (models.Submission, error)
	GradeJob(ctx context.Context, job queue.Job) error
}

type useCase struct {
	getProblem       problemUseCase.GetProblemUseCase
	createSubmission submissionUseCase.CreateSubmissionUseCase
	updateSubmission submissionUseCase.UpdateSubmissionUseCase
	batchResults     resultUseCase.BatchCreateTestCaseResultUseCase
	jobQueue         *queue.Queue
	grader           *runner.Runner
}

func NewUseCase(
	getProblem problemUseCase.GetProblemUseCase,
	createSubmission submissionUseCase.CreateSubmissionUseCase,
	updateSubmission submissionUseCase.UpdateSubmissionUseCase,
	batchResults resultUseCase.BatchCreateTestCaseResultUseCase,
	jobQueue *queue.Queue,
	grader *runner.Runner,
) UseCase {
	return &useCase{
		getProblem:       getProblem,
		createSubmission: createSubmission,
		updateSubmission: updateSubmission,
		batchResults:     batchResults,
		jobQueue:         jobQueue,
		grader:           grader,
	}
}

func (uc *useCase) Submit(ctx context.Context, input SubmitInput) (models.Submission, error) {
	problem, err := uc.getProblem.Execute(problemUseCase.GetProblemInput{ID: input.ProblemID})
	if err != nil {
		return models.Submission{}, err
	}

	submission, err := uc.createSubmission.Execute(submissionUseCase.CreateSubmissionInput{
		ProblemID:  problem.ID,
		UserID:     input.UserID,
		SourceCode: input.SourceCode,
	})
	if err != nil {
		return models.Submission{}, err
	}

	_, err = uc.jobQueue.Enqueue(ctx, queue.Job{
		ID:         submission.ID,
		Submission: submission,
		Problem:    problem,
		TestCases:  problem.TestCases,
	})
	if err != nil {
		message := "cannot enqueue grading job"
		_, _ = uc.updateSubmission.Execute(submissionUseCase.UpdateSubmissionInput{
			ID:           submission.ID,
			Status:       "queue_error",
			ErrorMessage: &message,
		})
		return models.Submission{}, err
	}

	return submission, nil
}

func (uc *useCase) GradeJob(ctx context.Context, job queue.Job) error {
	results, err := uc.grader.Run(ctx, &job.Submission, &job.Problem, job.TestCases)
	if err != nil {
		message := err.Error()
		_, updateErr := uc.updateSubmission.Execute(submissionUseCase.UpdateSubmissionInput{
			ID:           job.Submission.ID,
			Status:       "system_error",
			ErrorMessage: &message,
		})
		return errors.Join(err, updateErr)
	}

	passed := 0
	totalRuntime := time.Duration(0)
	dbResults := make([]models.TestCaseResult, 0, len(results))
	status := runner.VerdictAccepted
	for _, result := range results {
		if result.Passed {
			passed++
		} else if status == runner.VerdictAccepted {
			status = result.Verdict
		}
		totalRuntime += result.RunTime

		dbResult := models.TestCaseResult{
			ID:           uuid.New().String(),
			SubmissionID: job.Submission.ID,
			TestCaseID:   result.TestCaseID,
			IsPassed:     result.Passed,
			Verdict:      result.Verdict,
			RunTime:      int(result.RunTime.Milliseconds()),
		}
		if result.Error != nil {
			feedback := result.Error.Error()
			dbResult.Feedback = &feedback
		} else if result.Stderr != "" {
			feedback := result.Stderr
			dbResult.Feedback = &feedback
		}
		dbResults = append(dbResults, dbResult)
	}

	if err := uc.batchResults.Execute(resultUseCase.BatchCreateTestCaseResultInput{
		TestCaseResults: dbResults,
	}); err != nil {
		return err
	}

	score := 0
	if len(results) > 0 {
		score = passed * 100 / len(results)
	}
	_, err = uc.updateSubmission.Execute(submissionUseCase.UpdateSubmissionInput{
		ID:      job.Submission.ID,
		Status:  status,
		Score:   score,
		RunTime: int(totalRuntime.Milliseconds()),
	})
	return err
}
