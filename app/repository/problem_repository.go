package repository

import (
	"ddp0_grader/app/models"

	"gorm.io/gorm"
)

type ProblemRepository interface {
	GetProblemByID(id string) (*models.Problem, error)
	GetProblemByIDWithPreloaded(id string) (*models.Problem, error)
	GetAllProblems() ([]models.Problem, error)
	SaveProblem(problem *models.Problem) error
	DeleteProblem(problem *models.Problem) error
}

type problemRepository struct {
	db *gorm.DB
}

func NewProblemRepository(db *gorm.DB) ProblemRepository {
	return &problemRepository{db}
}

func (r *problemRepository) GetProblemByIDWithPreloaded(id string) (*models.Problem, error) {
	var problem models.Problem
	if err := r.db.Preload("TestCases").First(&problem, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &problem, nil
}

func (r *problemRepository) GetProblemByID(id string) (*models.Problem, error) {
	var problem models.Problem
	if err := r.db.First(&problem, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &problem, nil
}

func (r *problemRepository) GetAllProblems() ([]models.Problem, error) {
	var problems []models.Problem
	if err := r.db.Find(&problems).Error; err != nil {
		return nil, err
	}
	return problems, nil
}

func (r *problemRepository) SaveProblem(problem *models.Problem) error {
	if err := r.db.Save(problem).Error; err != nil {
		return err
	}
	return nil
}

func (r *problemRepository) DeleteProblem(problem *models.Problem) error {
	if err := r.db.Delete(problem).Error; err != nil {
		return err
	}
	return nil
}
