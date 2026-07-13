package testcase

import (
	"context"
	"testing"

	"ddp0_grader/app/models"

	"gorm.io/gorm"
)

type fakeProblemRepository struct{ problems map[string]models.Problem }

func (r *fakeProblemRepository) GetProblemByID(id string) (*models.Problem, error) {
	problem, ok := r.problems[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return &problem, nil
}
func (r *fakeProblemRepository) GetProblemByIDWithPreloaded(id string) (*models.Problem, error) {
	return r.GetProblemByID(id)
}
func (r *fakeProblemRepository) GetAllProblems() ([]models.Problem, error) { return nil, nil }
func (r *fakeProblemRepository) SaveProblem(*models.Problem) error         { return nil }
func (r *fakeProblemRepository) DeleteProblem(*models.Problem) error       { return nil }

type fakeTestCaseRepository struct{ items map[string]models.TestCase }

func (r *fakeTestCaseRepository) GetTestCaseByID(id string) (*models.TestCase, error) {
	testCase, ok := r.items[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return &testCase, nil
}
func (r *fakeTestCaseRepository) GetTestCasesByProblemID(problemID string) ([]models.TestCase, error) {
	items := []models.TestCase{}
	for _, testCase := range r.items {
		if testCase.ProblemID == problemID {
			items = append(items, testCase)
		}
	}
	return items, nil
}
func (r *fakeTestCaseRepository) SaveTestCase(testCase *models.TestCase) error {
	r.items[testCase.ID] = *testCase
	return nil
}
func (r *fakeTestCaseRepository) DeleteTestCase(testCase *models.TestCase) error {
	delete(r.items, testCase.ID)
	return nil
}

func TestUseCaseCRUD(t *testing.T) {
	problemRepo := &fakeProblemRepository{problems: map[string]models.Problem{"problem-1": {ID: "problem-1"}}}
	testCaseRepo := &fakeTestCaseRepository{items: make(map[string]models.TestCase)}
	useCase := NewUseCase(problemRepo, testCaseRepo)
	ctx := context.Background()

	created, err := useCase.Create(ctx, CreateInput{ProblemID: "problem-1", Input: "", Output: "42", IsHidden: true})
	if err != nil || created.ID == "" || created.Input != "" {
		t.Fatalf("Create() = (%+v, %v)", created, err)
	}

	updated, err := useCase.Update(ctx, created.ID, UpdateInput{Input: "1 41", Output: "42", IsHidden: false})
	if err != nil || updated.IsHidden || updated.Input != "1 41" {
		t.Fatalf("Update() = (%+v, %v)", updated, err)
	}

	items, err := useCase.GetByProblemID(ctx, "problem-1")
	if err != nil || len(items) != 1 {
		t.Fatalf("GetByProblemID() = (%d items, %v)", len(items), err)
	}

	if err := useCase.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
}

func TestUseCaseRejectsUnknownProblem(t *testing.T) {
	useCase := NewUseCase(&fakeProblemRepository{problems: map[string]models.Problem{}}, &fakeTestCaseRepository{items: map[string]models.TestCase{}})
	_, err := useCase.Create(context.Background(), CreateInput{ProblemID: "missing"})
	if err != gorm.ErrRecordNotFound {
		t.Fatalf("Create() error = %v, want record not found", err)
	}
}
