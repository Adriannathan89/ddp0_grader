package grading

import (
	"context"
	"errors"
	"time"

	"ddp0_grader/app/models"
	"ddp0_grader/app/repository"
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
	GetSubmission(ctx context.Context, id string) (models.Submission, error)
	GradeJob(ctx context.Context, job queue.Job) error
}

type JobQueue interface {
	Enqueue(ctx context.Context, job queue.Job) (string, error)
}

type Grader interface {
	Run(ctx context.Context, submission *models.Submission, problem *models.Problem, testCases []models.TestCase) ([]runner.TestResult, error)
}

type useCase struct {
	problemRepo    repository.ProblemRepository
	submissionRepo repository.SubmissionRepository
	resultRepo     repository.TestCaseResultRepository
	jobQueue       JobQueue
	grader         Grader
}

func NewUseCase(
	problemRepo repository.ProblemRepository,
	submissionRepo repository.SubmissionRepository,
	resultRepo repository.TestCaseResultRepository,
	jobQueue JobQueue,
	grader Grader,
) UseCase {
	return &useCase{
		problemRepo:    problemRepo,
		submissionRepo: submissionRepo,
		resultRepo:     resultRepo,
		jobQueue:       jobQueue,
		grader:         grader,
	}
}

func (uc *useCase) Submit(ctx context.Context, input SubmitInput) (models.Submission, error) {
	problem, err := uc.problemRepo.GetProblemByIDWithPreloaded(input.ProblemID)
	if err != nil {
		return models.Submission{}, err
	}

	submission := models.Submission{
		ID:         uuid.New().String(),
		ProblemID:  problem.ID,
		UserID:     input.UserID,
		SourceCode: input.SourceCode,
		Status:     models.SubmissionStatusQueued,
	}
	if err := uc.submissionRepo.SaveSubmission(&submission); err != nil {
		return models.Submission{}, err
	}

	_, err = uc.jobQueue.Enqueue(ctx, queue.Job{
		ID:         submission.ID,
		Submission: submission,
		Problem:    *problem,
		TestCases:  problem.TestCases,
	})
	if err != nil {
		return models.Submission{}, err
	}

	return submission, nil
}

func (uc *useCase) GetSubmission(_ context.Context, id string) (models.Submission, error) {
	submission, err := uc.submissionRepo.GetSubmissionByIDWithPreloaded(id)
	if err != nil {
		return models.Submission{}, err
	}

	return *submission, nil
}

func (uc *useCase) GradeJob(ctx context.Context, job queue.Job) error {
	results, err := uc.grader.Run(ctx, &job.Submission, &job.Problem, job.TestCases)
	if err != nil {
		job.Submission.Status = models.SubmissionStatusWrongAnswer
		updateErr := uc.submissionRepo.SaveSubmission(&job.Submission)
		return errors.Join(err, updateErr)
	}

	passed := 0
	totalRuntime := time.Duration(0)
	dbResults := make([]models.TestCaseResult, 0, len(results))
	status := models.SubmissionStatusAccepted
	for _, result := range results {
		if result.Passed {
			passed++
		} else if status == models.SubmissionStatusAccepted {
			status = submissionStatus(result.Verdict)
		}
		totalRuntime += result.RunTime

		dbResult := models.TestCaseResult{
			ID:           uuid.New().String(),
			SubmissionID: job.Submission.ID,
			TestCaseID:   result.TestCaseID,
			IsPassed:     result.Passed,
			Verdict:      result.Verdict,
			RunTime:      int(result.RunTime.Milliseconds()),
			MemoryUsage:  0,
		}
		if result.Error != nil {
			message := result.Error.Error()
			dbResult.ErrorMessage = &message
			dbResult.Feedback = &message
		} else if result.Stderr != "" {
			feedback := result.Stderr
			dbResult.Feedback = &feedback
		}
		dbResults = append(dbResults, dbResult)
	}

	if err := uc.resultRepo.BatchSaveTestCaseResults(dbResults); err != nil {
		return err
	}

	score := 0
	if len(results) > 0 {
		score = passed * 100 / len(results)
	}
	job.Submission.Status = status
	job.Submission.Score = score
	job.Submission.TotalRunTime = int(totalRuntime.Milliseconds())
	return uc.submissionRepo.SaveSubmission(&job.Submission)
}

func submissionStatus(verdict string) string {
	switch verdict {
	case runner.VerdictAccepted:
		return models.SubmissionStatusAccepted
	case runner.VerdictTimeLimitExceeded:
		return models.SubmissionStatusTimeLimitExceded
	case runner.VerdictOutputLimit:
		return models.SubmissionStatusMemoryLimitExceded
	default:
		return models.SubmissionStatusWrongAnswer
	}
}
