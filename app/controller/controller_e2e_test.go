package controller_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"ddp0_grader/app/config"
	"ddp0_grader/app/controller"
	"ddp0_grader/app/models"
	"ddp0_grader/app/usecase/grading"
	problemuc "ddp0_grader/app/usecase/problem"
	testcaseuc "ddp0_grader/app/usecase/testcase"
	"ddp0_grader/pkg/queue"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type problemFake struct{ item models.Problem }

func (f *problemFake) Create(_ context.Context, input problemuc.CreateInput) (models.Problem, error) {
	f.item = models.Problem{ID: "problem-1", Title: input.Title, Description: input.Description, Author: input.Author, Tag: input.Tag, Difficulty: input.Difficulty, TimeLimit: input.TimeLimit, MemoryLimit: input.MemoryLimit, Hint: input.Hint, IsQuiz: input.IsQuiz}
	return f.item, nil
}
func (f *problemFake) GetAll(_ context.Context, isQuiz *bool) ([]models.Problem, error) {
	if isQuiz != nil && f.item.IsQuiz != *isQuiz {
		return []models.Problem{}, nil
	}
	return []models.Problem{f.item}, nil
}
func (f *problemFake) GetByID(_ context.Context, id string) (models.Problem, error) {
	if id != f.item.ID {
		return models.Problem{}, gorm.ErrRecordNotFound
	}
	return f.item, nil
}
func (f *problemFake) Update(_ context.Context, id string, input problemuc.UpdateInput) (models.Problem, error) {
	if id != f.item.ID {
		return models.Problem{}, gorm.ErrRecordNotFound
	}
	f.item.Title, f.item.Description, f.item.Author = input.Title, input.Description, input.Author
	f.item.Tag, f.item.Difficulty = input.Tag, input.Difficulty
	f.item.TimeLimit, f.item.MemoryLimit, f.item.Hint, f.item.IsQuiz = input.TimeLimit, input.MemoryLimit, input.Hint, input.IsQuiz
	return f.item, nil
}
func (f *problemFake) Delete(_ context.Context, id string) error {
	if id != f.item.ID {
		return gorm.ErrRecordNotFound
	}
	f.item = models.Problem{}
	return nil
}

type testCaseFake struct{ item models.TestCase }

func (f *testCaseFake) Create(_ context.Context, input testcaseuc.CreateInput) (models.TestCase, error) {
	if input.ProblemID != "problem-1" {
		return models.TestCase{}, gorm.ErrRecordNotFound
	}
	f.item = models.TestCase{ID: "testcase-1", ProblemID: input.ProblemID, Input: input.Input, Output: input.Output, IsHidden: input.IsHidden}
	return f.item, nil
}
func (f *testCaseFake) GetByID(_ context.Context, id string) (models.TestCase, error) {
	if id != f.item.ID {
		return models.TestCase{}, gorm.ErrRecordNotFound
	}
	return f.item, nil
}
func (f *testCaseFake) GetByProblemID(_ context.Context, id string) ([]models.TestCase, error) {
	if id != "problem-1" {
		return nil, gorm.ErrRecordNotFound
	}
	return []models.TestCase{f.item}, nil
}
func (f *testCaseFake) GetAllByProblemID(ctx context.Context, id string) ([]models.TestCase, error) {
	return f.GetByProblemID(ctx, id)
}
func (f *testCaseFake) Update(_ context.Context, id string, input testcaseuc.UpdateInput) (models.TestCase, error) {
	if id != f.item.ID {
		return models.TestCase{}, gorm.ErrRecordNotFound
	}
	f.item.Input, f.item.Output, f.item.IsHidden = input.Input, input.Output, input.IsHidden
	return f.item, nil
}
func (f *testCaseFake) Delete(_ context.Context, id string) error {
	if id != f.item.ID {
		return gorm.ErrRecordNotFound
	}
	f.item = models.TestCase{}
	return nil
}

type gradingFake struct{ submission models.Submission }

func (f *gradingFake) Submit(_ context.Context, input grading.SubmitInput) (models.Submission, error) {
	if input.ProblemID == "missing" {
		return models.Submission{}, gorm.ErrRecordNotFound
	}
	f.submission = models.Submission{ID: "submission-1", ProgressID: "progress-1", Progress: models.Progress{ID: "progress-1", UserID: input.UserID}, SourceCode: input.SourceCode, Status: models.SubmissionStatusQueued}
	return f.submission, nil
}
func (f *gradingFake) GetSubmission(_ context.Context, id string) (models.Submission, error) {
	if id != f.submission.ID {
		return models.Submission{}, gorm.ErrRecordNotFound
	}
	return f.submission, nil
}
func (f *gradingFake) GradeJob(context.Context, queue.Job) error                { return nil }
func (f *gradingFake) MarkJobExhausted(context.Context, queue.Job, error) error { return nil }

type gradeRateLimiterFake struct {
	allowed    bool
	retryAfter time.Duration
	err        error
	calls      int
}

func (f *gradeRateLimiterFake) Allow(_ context.Context, userID string) (bool, time.Duration, error) {
	f.calls++
	if userID != "user-1" {
		return false, 0, errors.New("unexpected user")
	}
	return f.allowed, f.retryAfter, f.err
}

type progressFake struct {
	progress models.Progress
}

func (f *progressFake) GetByUserID(_ context.Context, userID string) ([]models.Progress, error) {
	if userID != "user-1" {
		return []models.Progress{}, nil
	}
	return []models.Progress{f.progress}, nil
}
func (f *progressFake) GetByUserAndProblemID(_ context.Context, userID, problemID string) (models.Progress, error) {
	if userID != "user-1" || problemID != "problem-1" {
		return models.Progress{}, gorm.ErrRecordNotFound
	}
	return f.progress, nil
}

func newRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := router.Group("/api")
	api.Use(func(c *gin.Context) {
		c.Set(config.AuthUserIDContextKey, "user-1")
		c.Set(config.AuthAccessTokenContextKey, "access-token")
		c.Next()
	})
	controller.NewHealthController().RegisterRoutes(api)
	controller.NewProblemController(&problemFake{}).RegisterRoutes(api)
	testCaseController := controller.NewTestCaseController(&testCaseFake{})
	testCaseController.RegisterRoutes(api)
	testCaseController.RegisterAdminRoutes(api)
	controller.NewSubmissionController(&gradingFake{}).RegisterRoutes(api)
	controller.NewProgressController(&progressFake{progress: models.Progress{ID: "progress-1", UserID: "user-1", ProblemID: "problem-1", Submissions: []models.Submission{{ID: "submission-1"}}}}).RegisterRoutes(api)
	return router
}

func request(router http.Handler, method, path, contentType string, body io.Reader) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, body)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	router.ServeHTTP(recorder, req)
	return recorder
}

func TestRoutesE2E(t *testing.T) {
	router := newRouter()

	if response := request(router, http.MethodGet, "/api/health", "", nil); response.Code != http.StatusOK {
		t.Fatalf("GET /api/health status = %d, want 200", response.Code)
	}

	problemBody := `{"title":"Sum","description":"Add two integers","created_by":"lecturer","tag":"math","difficulty":"easy","time_limit":2,"memory_limit":256,"hint":"Tambahkan dua nilai.","is_quiz":true}`
	if response := request(router, http.MethodPost, "/api/problems", "application/json", bytes.NewBufferString(problemBody)); response.Code != http.StatusCreated {
		t.Fatalf("POST /api/problems status = %d, body = %s", response.Code, response.Body.String())
	}
	if response := request(router, http.MethodGet, "/api/problems", "", nil); response.Code != http.StatusOK {
		t.Fatalf("GET /api/problems status = %d", response.Code)
	}
	if response := request(router, http.MethodGet, "/api/problems?is_quiz=false", "", nil); response.Code != http.StatusOK || response.Body.String() != "[]" {
		t.Fatalf("GET /api/problems?is_quiz=false = %d, body = %s", response.Code, response.Body.String())
	}
	if response := request(router, http.MethodGet, "/api/problems/problem-1", "", nil); response.Code != http.StatusOK {
		t.Fatalf("GET /api/problems/:id status = %d", response.Code)
	}
	updatedProblem := `{"title":"Sum v2","description":"Add values","created_by":"lecturer","tag":"sorting","difficulty":"medium","time_limit":3,"memory_limit":512,"is_quiz":false}`
	if response := request(router, http.MethodPatch, "/api/problems/problem-1", "application/json", bytes.NewBufferString(updatedProblem)); response.Code != http.StatusOK {
		t.Fatalf("PATCH /api/problems/:id status = %d, body = %s", response.Code, response.Body.String())
	}

	testCaseBody := `{"input":"1 2","output":"3","is_hidden":true}`
	if response := request(router, http.MethodPost, "/api/problems/problem-1/testcases", "application/json", bytes.NewBufferString(testCaseBody)); response.Code != http.StatusCreated {
		t.Fatalf("POST testcase status = %d, body = %s", response.Code, response.Body.String())
	}
	if response := request(router, http.MethodGet, "/api/problems/problem-1/testcases", "", nil); response.Code != http.StatusOK {
		t.Fatalf("GET testcases status = %d", response.Code)
	}
	if response := request(router, http.MethodGet, "/api/admin/problems/problem-1/testcases", "", nil); response.Code != http.StatusOK {
		t.Fatalf("GET admin testcases status = %d", response.Code)
	}
	if response := request(router, http.MethodGet, "/api/testcases/testcase-1", "", nil); response.Code != http.StatusOK {
		t.Fatalf("GET testcase status = %d", response.Code)
	}
	if response := request(router, http.MethodPatch, "/api/testcases/testcase-1", "application/json", bytes.NewBufferString(`{"input":"2 3","output":"5","is_hidden":false}`)); response.Code != http.StatusOK {
		t.Fatalf("PATCH testcase status = %d", response.Code)
	}

	multipartBody := &bytes.Buffer{}
	writer := multipart.NewWriter(multipartBody)
	_ = writer.WriteField("problem_id", "problem-1")
	part, err := writer.CreateFormFile("file", "solution.py")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = part.Write([]byte("print(3)"))
	_ = writer.Close()
	response := request(router, http.MethodPost, "/api/submissions/grade", writer.FormDataContentType(), multipartBody)
	if response.Code != http.StatusAccepted {
		t.Fatalf("POST /api/submissions/grade status = %d, body = %s", response.Code, response.Body.String())
	}
	var submitted map[string]string
	if err := json.Unmarshal(response.Body.Bytes(), &submitted); err != nil || submitted["submission_id"] != "submission-1" {
		t.Fatalf("submission response = %s, error = %v", response.Body.String(), err)
	}
	if response := request(router, http.MethodGet, "/api/submissions/submission-1", "", nil); response.Code != http.StatusOK {
		t.Fatalf("GET /submissions/:id status = %d", response.Code)
	}
	if response := request(router, http.MethodGet, "/api/progresses", "", nil); response.Code != http.StatusOK {
		t.Fatalf("GET /api/progresses status = %d", response.Code)
	}
	if response := request(router, http.MethodGet, "/api/progresses/problem-1", "", nil); response.Code != http.StatusOK {
		t.Fatalf("GET /api/progresses/:problemID status = %d", response.Code)
	}

	if response := request(router, http.MethodDelete, "/api/testcases/testcase-1", "", nil); response.Code != http.StatusNoContent {
		t.Fatalf("DELETE testcase status = %d", response.Code)
	}
	if response := request(router, http.MethodDelete, "/api/problems/problem-1", "", nil); response.Code != http.StatusNoContent {
		t.Fatalf("DELETE problem status = %d", response.Code)
	}
}

func TestRoutesReturnExpectedClientErrors(t *testing.T) {
	router := newRouter()

	if response := request(router, http.MethodPost, "/api/problems", "application/json", bytes.NewBufferString(`{`)); response.Code != http.StatusBadRequest {
		t.Fatalf("invalid problem JSON status = %d, want 400", response.Code)
	}
	if response := request(router, http.MethodGet, "/api/problems/missing", "", nil); response.Code != http.StatusNotFound {
		t.Fatalf("missing problem status = %d, want 404", response.Code)
	}
	if response := request(router, http.MethodPost, "/api/submissions/grade", "", nil); response.Code != http.StatusBadRequest {
		t.Fatalf("submission without form status = %d, want 400", response.Code)
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("problem_id", "problem-1")
	part, err := writer.CreateFormFile("file", "solution.txt")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = part.Write([]byte("print(3)"))
	_ = writer.Close()
	if response := request(router, http.MethodPost, "/api/submissions/grade", writer.FormDataContentType(), body); response.Code != http.StatusBadRequest {
		t.Fatalf("submission with non-python file status = %d, want 400", response.Code)
	}
}

func TestSubmissionGradeRateLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(config.AuthUserIDContextKey, "user-1")
		c.Next()
	})
	limiter := &gradeRateLimiterFake{allowed: false, retryAfter: 9*time.Second + time.Millisecond}
	controller.NewSubmissionController(&gradingFake{}, limiter).RegisterRoutes(router)

	response := request(router, http.MethodPost, "/submissions/grade", "", nil)
	if response.Code != http.StatusTooManyRequests {
		t.Fatalf("POST /submissions/grade status = %d, want 429", response.Code)
	}
	if got := response.Header().Get("Retry-After"); got != "10" {
		t.Fatalf("Retry-After = %q, want 10", got)
	}
	if limiter.calls != 1 {
		t.Fatalf("rate limiter calls = %d, want 1", limiter.calls)
	}

	// The Celery-only quiz endpoint is idempotent and must not be delayed by
	// the public submission rate limiter.
	response = request(router, http.MethodPost, "/quiz/submissions/grade", "", nil)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("POST /quiz/submissions/grade status = %d, want 400", response.Code)
	}
	if limiter.calls != 1 {
		t.Fatalf("quiz endpoint invoked rate limiter %d times, want 1", limiter.calls)
	}
}

func TestQuizSubmissionRequiresIdempotencyKey(t *testing.T) {
	router := newRouter()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("problem_id", "problem-1")
	part, err := writer.CreateFormFile("file", "solution.py")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = part.Write([]byte("print(3)"))
	_ = writer.Close()

	response := request(router, http.MethodPost, "/api/quiz/submissions/grade", writer.FormDataContentType(), body)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("POST quiz grade without idempotency key status = %d, body = %s", response.Code, response.Body.String())
	}

	body = &bytes.Buffer{}
	writer = multipart.NewWriter(body)
	_ = writer.WriteField("problem_id", "problem-1")
	part, err = writer.CreateFormFile("file", "solution.py")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = part.Write([]byte("print(3)"))
	_ = writer.Close()
	req := httptest.NewRequest(http.MethodPost, "/api/quiz/submissions/grade", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Idempotency-Key", "quiz-answer-1")
	response = httptest.NewRecorder()
	router.ServeHTTP(response, req)
	if response.Code != http.StatusAccepted {
		t.Fatalf("POST quiz grade with idempotency key status = %d, body = %s", response.Code, response.Body.String())
	}
}
