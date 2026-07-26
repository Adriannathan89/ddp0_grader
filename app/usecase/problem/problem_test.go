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

func (r *fakeRepository) GetAllProblems(isQuiz *bool) ([]models.Problem, error) {
	problems := make([]models.Problem, 0, len(r.items))
	for _, problem := range r.items {
		if isQuiz != nil && problem.IsQuiz != *isQuiz {
			continue
		}
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

	created, err := useCase.Create(ctx, CreateInput{Title: "Sum", Description: "Add two values", Author: "lecturer", Tag: models.TagMath, Difficulty: models.DifficultyEasy, TimeLimit: 2, MemoryLimit: 64, Hint: "Mulai dari variabel penampung."})
	if err != nil || created.ID == "" || created.Hint == "" {
		t.Fatalf("Create() = (%+v, %v), want created problem", created, err)
	}

	got, err := useCase.GetByID(ctx, created.ID)
	if err != nil || got.Title != "Sum" {
		t.Fatalf("GetByID() = (%+v, %v)", got, err)
	}

	updated, err := useCase.Update(ctx, created.ID, UpdateInput{Title: "Sum v2", Description: "Add", Author: "lecturer", Tag: "Data Structure", Difficulty: models.DifficultyMedium, TimeLimit: 3, MemoryLimit: 64, Hint: "Gunakan operator +.", IsQuiz: true})
	if err != nil || updated.TimeLimit != 3 || updated.Title != "Sum v2" || updated.Hint != "Gunakan operator +." || !updated.IsQuiz {
		t.Fatalf("Update() = (%+v, %v)", updated, err)
	}
	if updated.Tag != models.TagDataStructure {
		t.Fatalf("updated tag = %q, want %q", updated.Tag, models.TagDataStructure)
	}

	all, err := useCase.GetAll(ctx, nil)
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

func TestUseCaseFiltersProblemsByQuizFlag(t *testing.T) {
	repo := newFakeRepository()
	useCase := NewUseCase(repo)
	ctx := context.Background()

	if _, err := useCase.Create(ctx, CreateInput{Title: "Gym", Description: "Practice", Author: "lecturer", Tag: models.TagMath, Difficulty: models.DifficultyEasy, TimeLimit: 1, MemoryLimit: 1}); err != nil {
		t.Fatalf("create gym problem: %v", err)
	}
	if _, err := useCase.Create(ctx, CreateInput{Title: "Quiz", Description: "Assessment", Author: "lecturer", Tag: models.TagMath, Difficulty: models.DifficultyEasy, TimeLimit: 1, MemoryLimit: 1, IsQuiz: true}); err != nil {
		t.Fatalf("create quiz problem: %v", err)
	}

	isQuiz := true
	quizProblems, err := useCase.GetAll(ctx, &isQuiz)
	if err != nil || len(quizProblems) != 1 || !quizProblems[0].IsQuiz {
		t.Fatalf("GetAll(true) = (%+v, %v), want only quiz problem", quizProblems, err)
	}

	isQuiz = false
	gymProblems, err := useCase.GetAll(ctx, &isQuiz)
	if err != nil || len(gymProblems) != 1 || gymProblems[0].IsQuiz {
		t.Fatalf("GetAll(false) = (%+v, %v), want only gym problem", gymProblems, err)
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
