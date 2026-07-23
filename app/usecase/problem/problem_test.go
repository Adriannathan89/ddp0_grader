package problem

import (
	"context"
	"testing"

	"ddp0_grader/app/models"

	"gorm.io/gorm"
)

type fakeRepository struct {
	items map[string]models.Problem
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{items: make(map[string]models.Problem)}
}

func (r *fakeRepository) GetProblemByID(id string) (*models.Problem, error) {
	problem, ok := r.items[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return &problem, nil
}

func (r *fakeRepository) GetProblemByIDWithPreloaded(id string) (*models.Problem, error) {
	return r.GetProblemByID(id)
}

func (r *fakeRepository) GetAllProblems() ([]models.Problem, error) {
	problems := make([]models.Problem, 0, len(r.items))
	for _, problem := range r.items {
		problems = append(problems, problem)
	}
	return problems, nil
}

func (r *fakeRepository) SaveProblem(problem *models.Problem) error {
	r.items[problem.ID] = *problem
	return nil
}

func (r *fakeRepository) DeleteProblem(problem *models.Problem) error {
	delete(r.items, problem.ID)
	return nil
}

func TestUseCaseCRUD(t *testing.T) {
	repo := newFakeRepository()
	useCase := NewUseCase(repo)
	ctx := context.Background()

	created, err := useCase.Create(ctx, CreateInput{Title: "Sum", Description: "Add two values", Author: "lecturer", Tag: models.TagMath, Difficulty: models.DifficultyEasy, TimeLimit: 2, MemoryLimit: 64})
	if err != nil || created.ID == "" {
		t.Fatalf("Create() = (%+v, %v), want created problem", created, err)
	}

	got, err := useCase.GetByID(ctx, created.ID)
	if err != nil || got.Title != "Sum" {
		t.Fatalf("GetByID() = (%+v, %v)", got, err)
	}

	updated, err := useCase.Update(ctx, created.ID, UpdateInput{Title: "Sum v2", Description: "Add", Author: "lecturer", Tag: models.TagOperational, Difficulty: models.DifficultyMedium, TimeLimit: 3, MemoryLimit: 64})
	if err != nil || updated.TimeLimit != 3 || updated.Title != "Sum v2" {
		t.Fatalf("Update() = (%+v, %v)", updated, err)
	}

	all, err := useCase.GetAll(ctx)
	if err != nil || len(all) != 1 {
		t.Fatalf("GetAll() = (%d items, %v), want one item", len(all), err)
	}

	if err := useCase.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := useCase.GetByID(ctx, created.ID); err != gorm.ErrRecordNotFound {
		t.Fatalf("GetByID() after delete error = %v, want record not found", err)
	}
}

func TestUseCaseRejectsInvalidInput(t *testing.T) {
	useCase := NewUseCase(newFakeRepository())
	_, err := useCase.Create(context.Background(), CreateInput{Title: "", Description: "x", Author: "lecturer", Tag: models.TagMath, Difficulty: models.DifficultyEasy, TimeLimit: 1, MemoryLimit: 1})
	if err != ErrInvalidInput {
		t.Fatalf("Create() error = %v, want %v", err, ErrInvalidInput)
	}
}

func TestUseCaseRejectsLimitsAboveGraderCapacity(t *testing.T) {
	useCase := NewUseCase(newFakeRepository())
	valid := CreateInput{Title: "Sum", Description: "Add", Author: "lecturer", Tag: models.TagMath, Difficulty: models.DifficultyEasy, TimeLimit: 1_000, MemoryLimit: 64}
	if _, err := useCase.Create(context.Background(), valid); err != nil {
		t.Fatalf("Create() error = %v, want valid limits", err)
	}
	valid.TimeLimit = 1_001
	if _, err := useCase.Create(context.Background(), valid); err != ErrInvalidInput {
		t.Fatalf("Create() time limit error = %v, want %v", err, ErrInvalidInput)
	}
	valid.TimeLimit = 1_000
	valid.MemoryLimit = 65
	if _, err := useCase.Create(context.Background(), valid); err != ErrInvalidInput {
		t.Fatalf("Create() memory limit error = %v, want %v", err, ErrInvalidInput)
	}
}
