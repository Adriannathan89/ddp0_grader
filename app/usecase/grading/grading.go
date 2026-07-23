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
	"gorm.io/gorm"
)

type SubmitInput struct {
	ProblemID   string
	UserID      string
	AccessToken string
	SourceCode  string
}

const (
	maxTestCasesPerSubmission = 10
	maxFeedbackBytes          = 4 << 10
)

var ErrTooManyTestCases = errors.New("problem has too many testcases to grade")

type UseCase interface {
	Submit(ctx context.Context, input SubmitInput) (models.Submission, error)
	GetSubmission(ctx context.Context, id string) (models.Submission, error)
	GradeJob(ctx context.Context, job queue.Job) error
	MarkJobExhausted(ctx context.Context, job queue.Job, cause error) error
}

type JobQueue interface {
	Enqueue(ctx context.Context, job queue.Job) (string, error)
}

type Grader interface {
	Run(ctx context.Context, submission *models.Submission, problem *models.Problem, testCases []models.TestCase) ([]runner.TestResult, error)
}

// UserIdentityProvider obtains the current user from the identity service.
// It is called only when the user has not yet been stored locally.
type UserIdentityProvider interface {
	GetUser(ctx context.Context, accessToken string) (models.User, error)
}

type useCase struct {
	problemRepo      repository.ProblemRepository
	submissionRepo   repository.SubmissionRepository
	resultRepo       repository.TestCaseResultRepository
	progressRepo     repository.ProgressRepository
	userRepo         repository.UserRepository
	identityProvider UserIdentityProvider
	jobQueue         JobQueue
	grader           Grader
}

func NewUseCase(
	problemRepo repository.ProblemRepository,
	submissionRepo repository.SubmissionRepository,
	resultRepo repository.TestCaseResultRepository,
	progressRepo repository.ProgressRepository,
	userRepo repository.UserRepository,
	identityProvider UserIdentityProvider,
	jobQueue JobQueue,
	grader Grader,
) UseCase {
	return &useCase{
		problemRepo:      problemRepo,
		submissionRepo:   submissionRepo,
		resultRepo:       resultRepo,
		progressRepo:     progressRepo,
		userRepo:         userRepo,
		identityProvider: identityProvider,
		jobQueue:         jobQueue,
		grader:           grader,
	}
}

func (uc *useCase) Submit(ctx context.Context, input SubmitInput) (models.Submission, error) {
	problem, err := uc.problemRepo.GetProblemByIDWithPreloaded(input.ProblemID)
	if err != nil {
		return models.Submission{}, err
	}
	if len(problem.TestCases) > maxTestCasesPerSubmission {
		return models.Submission{}, ErrTooManyTestCases
	}
	if err := uc.ensureUser(ctx, input.UserID, input.AccessToken); err != nil {
		return models.Submission{}, err
	}
	existingProgress, err := uc.progressRepo.GetProgressByProblemAndUser(input.ProblemID, input.UserID)

	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return models.Submission{}, err
		}
		existingProgress = &models.Progress{
			ID:        uuid.New().String(),
			ProblemID: input.ProblemID,
			UserID:    input.UserID,
			BestScore: 0,
		}
		if err := uc.progressRepo.SaveProgress(existingProgress); err != nil {
			return models.Submission{}, err
		}
	}

	submission := models.Submission{
		ID:         uuid.New().String(),
		ProgressID: existingProgress.ID,
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
		submission.Status = models.SubmissionStatusSystemError
		return models.Submission{}, errors.Join(err, uc.submissionRepo.SaveSubmission(&submission))
	}

	return submission, nil
}

func (uc *useCase) ensureUser(ctx context.Context, userID, accessToken string) error {
	if _, err := uc.userRepo.GetUserByID(userID); err == nil {
		return nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if uc.identityProvider == nil {
		return errors.New("user identity provider is not configured")
	}

	user, err := uc.identityProvider.GetUser(ctx, accessToken)
	if err != nil {
		return err
	}
	if user.ID != userID {
		return errors.New("Django user id does not match authenticated user")
	}
	if err := uc.userRepo.SaveUser(&user); err != nil {
		// A concurrent submission may have inserted the same user already.
		if _, lookupErr := uc.userRepo.GetUserByID(userID); lookupErr == nil {
			return nil
		}
		return err
	}
	return nil
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
		job.Submission.Status = models.SubmissionStatusSystemError
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
			message := truncateFeedback(result.Error.Error())
			dbResult.ErrorMessage = &message
			dbResult.Feedback = &message
		} else if result.Stderr != "" {
			feedback := truncateFeedback(result.Stderr)
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
	if err := uc.progressRepo.UpdateBestScore(job.Submission.ProgressID, score); err != nil {
		return err
	}

	return uc.submissionRepo.SaveSubmission(&job.Submission)
}

func truncateFeedback(value string) string {
	if len(value) > maxFeedbackBytes {
		return value[:maxFeedbackBytes]
	}
	return value
}

// MarkJobExhausted records the terminal infrastructure failure before Queue
// acknowledges the message and moves it to the dead-letter stream.
func (uc *useCase) MarkJobExhausted(_ context.Context, job queue.Job, _ error) error {
	job.Submission.Status = models.SubmissionStatusSystemError
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
