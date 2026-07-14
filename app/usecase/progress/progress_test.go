package progress

import (
	"context"
	"testing"

	"ddp0_grader/app/models"

	"gorm.io/gorm"
)

type fakeRepository struct {
	progresses []models.Progress
}

func (r *fakeRepository) GetProgressByID(id string) (*models.Progress, error) {
	for _, progress := range r.progresses {
		if progress.ID == id {
			return &progress, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}
func (r *fakeRepository) GetProgressByProblemAndUser(problemID, userID string) (*models.Progress, error) {
	return r.GetProgressWithSubmissionsByProblemAndUser(problemID, userID)
}
func (r *fakeRepository) GetProgressWithSubmissionsByProblemAndUser(problemID, userID string) (*models.Progress, error) {
	for _, progress := range r.progresses {
		if progress.ProblemID == problemID && progress.UserID == userID {
			return &progress, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}
func (r *fakeRepository) GetProgressesWithSubmissionsByUserID(userID string) ([]models.Progress, error) {
	progresses := []models.Progress{}
	for _, progress := range r.progresses {
		if progress.UserID == userID {
			progresses = append(progresses, progress)
		}
	}
	return progresses, nil
}
func (r *fakeRepository) UpdateBestScore(string, int) error     { return nil }
func (r *fakeRepository) SaveProgress(*models.Progress) error   { return nil }
func (r *fakeRepository) DeleteProgress(*models.Progress) error { return nil }

func TestUseCaseGetsProgressesWithSubmissions(t *testing.T) {
	repo := &fakeRepository{progresses: []models.Progress{
		{ID: "progress-1", UserID: "user-1", ProblemID: "problem-1", Submissions: []models.Submission{{ID: "submission-1"}}},
		{ID: "progress-2", UserID: "user-1", ProblemID: "problem-2"},
	}}
	useCase := NewUseCase(repo)

	progresses, err := useCase.GetByUserID(context.Background(), "user-1")
	if err != nil || len(progresses) != 2 || len(progresses[0].Submissions) != 1 {
		t.Fatalf("GetByUserID() = (%+v, %v)", progresses, err)
	}

	progress, err := useCase.GetByUserAndProblemID(context.Background(), "user-1", "problem-1")
	if err != nil || progress.ID != "progress-1" || len(progress.Submissions) != 1 {
		t.Fatalf("GetByUserAndProblemID() = (%+v, %v)", progress, err)
	}
}

func TestUseCaseRejectsEmptyIdentifiers(t *testing.T) {
	useCase := NewUseCase(&fakeRepository{})
	if _, err := useCase.GetByUserID(context.Background(), " "); err != ErrInvalidInput {
		t.Fatalf("GetByUserID() error = %v, want %v", err, ErrInvalidInput)
	}
	if _, err := useCase.GetByUserAndProblemID(context.Background(), "user-1", " "); err != ErrInvalidInput {
		t.Fatalf("GetByUserAndProblemID() error = %v, want %v", err, ErrInvalidInput)
	}
}
