package grading

import (
	"context"
	"errors"
	"testing"
	"time"

	"ddp0_grader/app/models"
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

func (g *fakeGrader) Run(context.Context, *models.Submission, *models.Problem, []models.TestCase) ([]runner.TestResult, error) {
	return g.results, g.err
}

func TestSubmitQueuesSubmission(t *testing.T) {
	problem := models.Problem{ID: "problem-1", TestCases: []models.TestCase{{ID: "tc-1", ProblemID: "problem-1"}}}
	submissions := &fakeSubmissionRepository{items: map[string]models.Submission{}}
	jobQueue := &fakeQueue{}
	useCase := NewUseCase(&fakeProblemRepository{problem: problem}, submissions, &fakeResultRepository{}, jobQueue, &fakeGrader{})

	submission, err := useCase.Submit(context.Background(), SubmitInput{ProblemID: "problem-1", UserID: "user-1", SourceCode: "print(1)"})
	if err != nil {
		t.Fatalf("Submit() error = %v", err)
	}
	if submission.ID == "" || submission.Status != "queued" || jobQueue.job.ID != submission.ID {
		t.Fatalf("Submit() submission = %+v, queued job = %+v", submission, jobQueue.job)
	}
}

func TestSubmitMarksQueueError(t *testing.T) {
	problem := models.Problem{ID: "problem-1"}
	submissions := &fakeSubmissionRepository{items: map[string]models.Submission{}}
	useCase := NewUseCase(&fakeProblemRepository{problem: problem}, submissions, &fakeResultRepository{}, &fakeQueue{err: errors.New("redis unavailable")}, &fakeGrader{})

	_, err := useCase.Submit(context.Background(), SubmitInput{ProblemID: "problem-1", UserID: "user-1", SourceCode: "print(1)"})
	if err == nil || submissions.last.Status != "queue_error" || submissions.last.ErrorMessage == nil {
		t.Fatalf("Submit() error = %v, saved submission = %+v", err, submissions.last)
	}
}

func TestGradeJobSavesBatchResultsAndSubmission(t *testing.T) {
	submissions := &fakeSubmissionRepository{items: map[string]models.Submission{}}
	results := &fakeResultRepository{}
	grader := &fakeGrader{results: []runner.TestResult{
		{TestCaseID: "tc-1", Passed: true, Verdict: runner.VerdictAccepted, RunTime: 12 * time.Millisecond},
		{TestCaseID: "tc-2", Passed: false, Verdict: runner.VerdictWrongAnswer, RunTime: 8 * time.Millisecond, Stderr: "expected 2"},
	}}
	useCase := NewUseCase(&fakeProblemRepository{}, submissions, results, &fakeQueue{}, grader)
	job := queue.Job{Submission: models.Submission{ID: "submission-1", Status: "queued"}, Problem: models.Problem{ID: "problem-1"}, TestCases: []models.TestCase{{ID: "tc-1"}, {ID: "tc-2"}}}

	if err := useCase.GradeJob(context.Background(), job); err != nil {
		t.Fatalf("GradeJob() error = %v", err)
	}
	if len(results.saved) != 2 || results.saved[1].Feedback == nil || *results.saved[1].Feedback != "expected 2" {
		t.Fatalf("saved results = %+v", results.saved)
	}
	if submissions.last.Status != runner.VerdictWrongAnswer || submissions.last.Score != 50 || submissions.last.RunTime != 20 {
		t.Fatalf("saved submission = %+v", submissions.last)
	}
}

func TestGradeJobMarksSystemError(t *testing.T) {
	submissions := &fakeSubmissionRepository{items: map[string]models.Submission{}}
	useCase := NewUseCase(&fakeProblemRepository{}, submissions, &fakeResultRepository{}, &fakeQueue{}, &fakeGrader{err: errors.New("runner unavailable")})
	err := useCase.GradeJob(context.Background(), queue.Job{Submission: models.Submission{ID: "submission-1"}})
	if err == nil || submissions.last.Status != "system_error" || submissions.last.ErrorMessage == nil {
		t.Fatalf("GradeJob() error = %v, saved submission = %+v", err, submissions.last)
	}
}
