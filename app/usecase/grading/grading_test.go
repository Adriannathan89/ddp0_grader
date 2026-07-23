package grading

import (
	"context"
	"errors"
	"testing"
	"time"

	"ddp0_grader/app/models"
	"ddp0_grader/app/repository"
	"ddp0_grader/pkg/queue"
	"ddp0_grader/pkg/runner"

	"gorm.io/gorm"
)

type fakeProblemRepository struct{ problem models.Problem }

func (r *fakeProblemRepository) GetProblemByID(id string) (*models.Problem, error) {
	if id != r.problem.ID {
		return nil, gorm.ErrRecordNotFound
	}
	return &r.problem, nil
}
func (r *fakeProblemRepository) GetProblemByIDWithPreloaded(id string) (*models.Problem, error) {
	return r.GetProblemByID(id)
}
func (r *fakeProblemRepository) GetAllProblems() ([]models.Problem, error) { return nil, nil }
func (r *fakeProblemRepository) SaveProblem(*models.Problem) error         { return nil }
func (r *fakeProblemRepository) DeleteProblem(*models.Problem) error       { return nil }

type fakeSubmissionRepository struct {
	items map[string]models.Submission
	last  models.Submission
}

func (r *fakeSubmissionRepository) GetSubmissionByID(id string) (*models.Submission, error) {
	return r.GetSubmissionByIDWithPreloaded(id)
}
func (r *fakeSubmissionRepository) GetSubmissionByIDWithPreloaded(id string) (*models.Submission, error) {
	submission, ok := r.items[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return &submission, nil
}
func (r *fakeSubmissionRepository) SaveSubmission(submission *models.Submission) error {
	r.last = *submission
	r.items[submission.ID] = *submission
	return nil
}
func (r *fakeSubmissionRepository) DeleteSubmission(submission *models.Submission) error {
	delete(r.items, submission.ID)
	return nil
}
func (r *fakeSubmissionRepository) GetAdminSubmissions(repository.AdminSubmissionFilter) ([]repository.AdminSubmission, int64, error) {
	return nil, 0, nil
}

type fakeResultRepository struct{ saved []models.TestCaseResult }

func (r *fakeResultRepository) GetTestCaseResultByID(string) (*models.TestCaseResult, error) {
	return nil, gorm.ErrRecordNotFound
}
func (r *fakeResultRepository) GetTestCaseResultsBySubmissionID(string) ([]models.TestCaseResult, error) {
	return nil, nil
}
func (r *fakeResultRepository) GetTestCasesByUserIDAndSubmissionID(string, string) ([]models.TestCaseResult, error) {
	return nil, nil
}
func (r *fakeResultRepository) SaveTestCaseResult(*models.TestCaseResult) error { return nil }
func (r *fakeResultRepository) BatchSaveTestCaseResults(results []models.TestCaseResult) error {
	r.saved = append(r.saved, results...)
	return nil
}
func (r *fakeResultRepository) DeleteTestCaseResult(*models.TestCaseResult) error { return nil }

type fakeProgressRepository struct{ progress *models.Progress }

func (r *fakeProgressRepository) GetProgressByID(id string) (*models.Progress, error) {
	if r.progress == nil || r.progress.ID != id {
		return nil, gorm.ErrRecordNotFound
	}
	return r.progress, nil
}
func (r *fakeProgressRepository) GetProgressByProblemAndUser(problemID, userID string) (*models.Progress, error) {
	if r.progress == nil || r.progress.ProblemID != problemID || r.progress.UserID != userID {
		return nil, gorm.ErrRecordNotFound
	}
	return r.progress, nil
}
func (r *fakeProgressRepository) GetProgressWithSubmissionsByProblemAndUser(problemID, userID string) (*models.Progress, error) {
	return r.GetProgressByProblemAndUser(problemID, userID)
}
func (r *fakeProgressRepository) GetProgressesWithSubmissionsByUserID(userID string) ([]models.Progress, error) {
	if r.progress == nil || r.progress.UserID != userID {
		return []models.Progress{}, nil
	}
	return []models.Progress{*r.progress}, nil
}
func (r *fakeProgressRepository) UpdateBestScore(id string, score int) error {
	if r.progress == nil || r.progress.ID != id {
		return gorm.ErrRecordNotFound
	}
	if score > r.progress.BestScore {
		r.progress.BestScore = score
	}
	return nil
}
func (r *fakeProgressRepository) SaveProgress(progress *models.Progress) error {
	copy := *progress
	r.progress = &copy
	return nil
}
func (r *fakeProgressRepository) DeleteProgress(progress *models.Progress) error {
	if r.progress != nil && r.progress.ID == progress.ID {
		r.progress = nil
	}
	return nil
}

type fakeUserRepository struct{ users map[string]models.User }

func (r *fakeUserRepository) GetUserByID(id string) (*models.User, error) {
	user, ok := r.users[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return &user, nil
}
func (r *fakeUserRepository) GetUserByEmail(email string) (*models.User, error) {
	for _, user := range r.users {
		if user.Email == email {
			return &user, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}
func (r *fakeUserRepository) SaveUser(user *models.User) error {
	r.users[user.ID] = *user
	return nil
}
func (r *fakeUserRepository) DeleteUser(user *models.User) error {
	delete(r.users, user.ID)
	return nil
}

type fakeQueue struct {
	job queue.Job
	err error
}

func (q *fakeQueue) Enqueue(_ context.Context, job queue.Job) (string, error) {
	q.job = job
	if q.err != nil {
		return "", q.err
	}
	return "message-1", nil
}

type fakeGrader struct {
	results []runner.TestResult
	err     error
}

type fakeIdentityProvider struct {
	user models.User
	err  error
}

func (provider *fakeIdentityProvider) GetUser(context.Context, string) (models.User, error) {
	return provider.user, provider.err
}

func (g *fakeGrader) Run(context.Context, *models.Submission, *models.Problem, []models.TestCase) ([]runner.TestResult, error) {
	return g.results, g.err
}

func TestSubmitQueuesSubmission(t *testing.T) {
	problem := models.Problem{ID: "problem-1", TestCases: []models.TestCase{{ID: "tc-1", ProblemID: "problem-1"}}}
	submissions := &fakeSubmissionRepository{items: map[string]models.Submission{}}
	jobQueue := &fakeQueue{}
	progresses := &fakeProgressRepository{}
	users := &fakeUserRepository{users: map[string]models.User{"user-1": {ID: "user-1", Email: "user@example.com"}}}
	useCase := NewUseCase(&fakeProblemRepository{problem: problem}, submissions, &fakeResultRepository{}, progresses, users, nil, jobQueue, &fakeGrader{})

	submission, err := useCase.Submit(context.Background(), SubmitInput{ProblemID: "problem-1", UserID: "user-1", SourceCode: "print(1)"})
	if err != nil {
		t.Fatalf("Submit() error = %v", err)
	}
	if submission.ID == "" || submission.ProgressID == "" || submission.Status != models.SubmissionStatusQueued || jobQueue.job.ID != submission.ID || progresses.progress == nil {
		t.Fatalf("Submit() submission = %+v, queued job = %+v", submission, jobQueue.job)
	}
}

func TestSubmitRejectsMoreThanTenTestCases(t *testing.T) {
	testCases := make([]models.TestCase, maxTestCasesPerSubmission+1)
	for i := range testCases {
		testCases[i] = models.TestCase{ID: "tc"}
	}
	problem := models.Problem{ID: "problem-1", TestCases: testCases}
	useCase := NewUseCase(&fakeProblemRepository{problem: problem}, &fakeSubmissionRepository{items: map[string]models.Submission{}}, &fakeResultRepository{}, &fakeProgressRepository{}, &fakeUserRepository{users: map[string]models.User{"user-1": {ID: "user-1"}}}, nil, &fakeQueue{}, &fakeGrader{})
	_, err := useCase.Submit(context.Background(), SubmitInput{ProblemID: problem.ID, UserID: "user-1", SourceCode: "print(1)"})
	if !errors.Is(err, ErrTooManyTestCases) {
		t.Fatalf("Submit() error = %v, want %v", err, ErrTooManyTestCases)
	}
}

func TestSubmitMarksQueueError(t *testing.T) {
	problem := models.Problem{ID: "problem-1"}
	submissions := &fakeSubmissionRepository{items: map[string]models.Submission{}}
	progresses := &fakeProgressRepository{}
	users := &fakeUserRepository{users: map[string]models.User{"user-1": {ID: "user-1"}}}
	useCase := NewUseCase(&fakeProblemRepository{problem: problem}, submissions, &fakeResultRepository{}, progresses, users, nil, &fakeQueue{err: errors.New("redis unavailable")}, &fakeGrader{})

	_, err := useCase.Submit(context.Background(), SubmitInput{ProblemID: "problem-1", UserID: "user-1", SourceCode: "print(1)"})
	if err == nil || submissions.last.Status != models.SubmissionStatusSystemError {
		t.Fatalf("Submit() error = %v, saved submission = %+v", err, submissions.last)
	}
}

func TestGradeJobSavesBatchResultsAndSubmission(t *testing.T) {
	submissions := &fakeSubmissionRepository{items: map[string]models.Submission{}}
	results := &fakeResultRepository{}
	progresses := &fakeProgressRepository{progress: &models.Progress{ID: "progress-1"}}
	grader := &fakeGrader{results: []runner.TestResult{
		{TestCaseID: "tc-1", Passed: true, Verdict: runner.VerdictAccepted, RunTime: 12 * time.Millisecond},
		{TestCaseID: "tc-2", Passed: false, Verdict: runner.VerdictWrongAnswer, RunTime: 8 * time.Millisecond, Stderr: "expected 2"},
	}}
	useCase := NewUseCase(&fakeProblemRepository{}, submissions, results, progresses, &fakeUserRepository{users: map[string]models.User{}}, nil, &fakeQueue{}, grader)
	job := queue.Job{Submission: models.Submission{ID: "submission-1", ProgressID: "progress-1", Status: models.SubmissionStatusQueued}, Problem: models.Problem{ID: "problem-1"}, TestCases: []models.TestCase{{ID: "tc-1"}, {ID: "tc-2"}}}

	if err := useCase.GradeJob(context.Background(), job); err != nil {
		t.Fatalf("GradeJob() error = %v", err)
	}
	if len(results.saved) != 2 || results.saved[1].Feedback == nil || *results.saved[1].Feedback != "expected 2" || results.saved[1].MemoryUsage != 0 {
		t.Fatalf("saved results = %+v", results.saved)
	}
	if submissions.last.Status != models.SubmissionStatusWrongAnswer || submissions.last.Score != 50 || submissions.last.TotalRunTime != 20 {
		t.Fatalf("saved submission = %+v", submissions.last)
	}
}

func TestGradeJobMarksSystemError(t *testing.T) {
	submissions := &fakeSubmissionRepository{items: map[string]models.Submission{}}
	useCase := NewUseCase(&fakeProblemRepository{}, submissions, &fakeResultRepository{}, &fakeProgressRepository{}, &fakeUserRepository{users: map[string]models.User{}}, nil, &fakeQueue{}, &fakeGrader{err: errors.New("runner unavailable")})
	err := useCase.GradeJob(context.Background(), queue.Job{Submission: models.Submission{ID: "submission-1"}})
	if err == nil || submissions.last.Status != models.SubmissionStatusSystemError {
		t.Fatalf("GradeJob() error = %v, saved submission = %+v", err, submissions.last)
	}
}

func TestMarkJobExhaustedMarksSubmissionSystemError(t *testing.T) {
	submissions := &fakeSubmissionRepository{items: map[string]models.Submission{}}
	useCase := NewUseCase(&fakeProblemRepository{}, submissions, &fakeResultRepository{}, &fakeProgressRepository{}, &fakeUserRepository{users: map[string]models.User{}}, nil, &fakeQueue{}, &fakeGrader{})
	job := queue.Job{Submission: models.Submission{ID: "submission-1", Status: models.SubmissionStatusQueued}}

	if err := useCase.MarkJobExhausted(context.Background(), job, errors.New("database unavailable")); err != nil {
		t.Fatalf("MarkJobExhausted() error = %v", err)
	}
	if submissions.last.Status != models.SubmissionStatusSystemError {
		t.Fatalf("saved submission status = %q", submissions.last.Status)
	}
}

func TestSubmitCreatesMissingUserFromIdentityProvider(t *testing.T) {
	problem := models.Problem{ID: "problem-1"}
	users := &fakeUserRepository{users: map[string]models.User{}}
	identityProvider := &fakeIdentityProvider{user: models.User{ID: "user-1", Email: "user@example.com"}}
	useCase := NewUseCase(&fakeProblemRepository{problem: problem}, &fakeSubmissionRepository{items: map[string]models.Submission{}}, &fakeResultRepository{}, &fakeProgressRepository{}, users, identityProvider, &fakeQueue{}, &fakeGrader{})

	if _, err := useCase.Submit(context.Background(), SubmitInput{ProblemID: "problem-1", UserID: "user-1", AccessToken: "access-token", SourceCode: "print(1)"}); err != nil {
		t.Fatalf("Submit() error = %v", err)
	}
	if user, ok := users.users["user-1"]; !ok || user.Email != "user@example.com" {
		t.Fatalf("created user = %#v, exists = %t", user, ok)
	}
}

func TestSubmitRejectsMismatchedIdentityUser(t *testing.T) {
	problem := models.Problem{ID: "problem-1"}
	identityProvider := &fakeIdentityProvider{user: models.User{ID: "another-user", Email: "user@example.com"}}
	useCase := NewUseCase(&fakeProblemRepository{problem: problem}, &fakeSubmissionRepository{items: map[string]models.Submission{}}, &fakeResultRepository{}, &fakeProgressRepository{}, &fakeUserRepository{users: map[string]models.User{}}, identityProvider, &fakeQueue{}, &fakeGrader{})

	_, err := useCase.Submit(context.Background(), SubmitInput{ProblemID: "problem-1", UserID: "user-1", AccessToken: "access-token", SourceCode: "print(1)"})
	if err == nil || err.Error() != "Django user id does not match authenticated user" {
		t.Fatalf("Submit() error = %v", err)
	}
}

func TestSubmissionStatusNormalizesRunnerVerdicts(t *testing.T) {
	tests := map[string]string{
		runner.VerdictAccepted:          models.SubmissionStatusAccepted,
		runner.VerdictWrongAnswer:       models.SubmissionStatusWrongAnswer,
		runner.VerdictRuntimeError:      models.SubmissionStatusWrongAnswer,
		runner.VerdictSystemError:       models.SubmissionStatusWrongAnswer,
		runner.VerdictTimeLimitExceeded: models.SubmissionStatusTimeLimitExceded,
		runner.VerdictOutputLimit:       models.SubmissionStatusMemoryLimitExceded,
	}
	for verdict, want := range tests {
		if got := submissionStatus(verdict); got != want {
			t.Errorf("submissionStatus(%q) = %q, want %q", verdict, got, want)
		}
	}
}
