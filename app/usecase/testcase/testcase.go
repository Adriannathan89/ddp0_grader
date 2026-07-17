package testcase

import (
	"context"
	"errors"
	"strings"

	"ddp0_grader/app/models"
	"ddp0_grader/app/repository"

	"github.com/google/uuid"
)

var ErrInvalidInput = errors.New("invalid testcase input")

type CreateInput struct {
	ProblemID string
	Input     string
	Output    string
	IsHidden  bool
}

type UpdateInput struct {
	Input    string
	Output   string
	IsHidden bool
}

type UseCase interface {
	Create(ctx context.Context, input CreateInput) (models.TestCase, error)
	GetByID(ctx context.Context, id string) (models.TestCase, error)
	GetByProblemID(ctx context.Context, problemID string) ([]models.TestCase, error)
	GetAllByProblemID(ctx context.Context, problemID string) ([]models.TestCase, error)
	Update(ctx context.Context, id string, input UpdateInput) (models.TestCase, error)
	Delete(ctx context.Context, id string) error
}

type useCase struct {
	problemRepo  repository.ProblemRepository
	testCaseRepo repository.TestCaseRepository
}

func NewUseCase(problemRepo repository.ProblemRepository, testCaseRepo repository.TestCaseRepository) UseCase {
	return &useCase{problemRepo: problemRepo, testCaseRepo: testCaseRepo}
}

func (uc *useCase) Create(_ context.Context, input CreateInput) (models.TestCase, error) {
	problemID := strings.TrimSpace(input.ProblemID)
	if problemID == "" {
		return models.TestCase{}, ErrInvalidInput
	}
	if _, err := uc.problemRepo.GetProblemByID(problemID); err != nil {
		return models.TestCase{}, err
	}

	testCase := models.TestCase{ID: uuid.NewString(), ProblemID: problemID}
	if err := applyInput(&testCase, input.Input, input.Output, input.IsHidden); err != nil {
		return models.TestCase{}, err
	}
	if err := uc.testCaseRepo.SaveTestCase(&testCase); err != nil {
		return models.TestCase{}, err
	}
	return testCase, nil
}

func (uc *useCase) GetByID(_ context.Context, id string) (models.TestCase, error) {
	testCase, err := uc.testCaseRepo.GetTestCaseByID(strings.TrimSpace(id))
	if err != nil {
		return models.TestCase{}, err
	}
	return *testCase, nil
}

func (uc *useCase) GetByProblemID(ctx context.Context, problemID string) ([]models.TestCase, error) {
	testCases, err := uc.GetAllByProblemID(ctx, problemID)
	if err != nil {
		return nil, err
	}

	testCases = removePrivateTestCase(testCases)

	return testCases, nil
}

// GetAllByProblemID is intentionally reserved for the Django-admin protected
// route. Unlike the participant-facing method, it retains hidden input/output.
func (uc *useCase) GetAllByProblemID(_ context.Context, problemID string) ([]models.TestCase, error) {
	return uc.testCaseRepo.GetTestCasesByProblemID(strings.TrimSpace(problemID))
}

func removePrivateTestCase(testCases []models.TestCase) []models.TestCase {
	publicTestCases := make([]models.TestCase, 0, len(testCases))
	for _, testCase := range testCases {
		if !testCase.IsHidden {
			publicTestCases = append(publicTestCases, testCase)
		} else {
			publicTestCase := testCase
			publicTestCase.Input = ""
			publicTestCase.Output = ""
			publicTestCases = append(publicTestCases, publicTestCase)
		}
	}
	return publicTestCases
}

func (uc *useCase) Update(_ context.Context, id string, input UpdateInput) (models.TestCase, error) {
	testCase, err := uc.testCaseRepo.GetTestCaseByID(strings.TrimSpace(id))
	if err != nil {
		return models.TestCase{}, err
	}
	if err := applyInput(testCase, input.Input, input.Output, input.IsHidden); err != nil {
		return models.TestCase{}, err
	}
	if err := uc.testCaseRepo.SaveTestCase(testCase); err != nil {
		return models.TestCase{}, err
	}
	return *testCase, nil
}

func (uc *useCase) Delete(_ context.Context, id string) error {
	testCase, err := uc.testCaseRepo.GetTestCaseByID(strings.TrimSpace(id))
	if err != nil {
		return err
	}
	return uc.testCaseRepo.DeleteTestCase(testCase)
}

func applyInput(testCase *models.TestCase, input, output string, isHidden bool) error {
	testCase.Input = input
	testCase.Output = output
	testCase.IsHidden = isHidden
	return nil
}
