package repository

import (
	"ddp0_grader/app/models"
	"log"

	"gorm.io/gorm"
)

type ProgressRepository interface {
	GetProgressByID(id string) (*models.Progress, error)
	GetProgressByProblemAndUser(problemID string, userID string) (*models.Progress, error)
	GetProgressWithSubmissionsByProblemAndUser(problemID string, userID string) (*models.Progress, error)
	GetProgressesWithSubmissionsByUserID(userID string) ([]models.Progress, error)
	UpdateBestScore(progressID string, newScore int) error
	SaveProgress(progress *models.Progress) error
	DeleteProgress(progress *models.Progress) error
}

type progressRepository struct {
	db *gorm.DB
}

func NewProgressRepository(db *gorm.DB) ProgressRepository {
	return &progressRepository{db}
}

func (r *progressRepository) GetProgressByID(id string) (*models.Progress, error) {
	var progress models.Progress
	if err := r.db.Preload("Submissions").First(&progress, "id = ?", id).Error; err != nil {
		log.Printf("Error retrieving progress by ID: %v", err)
		return nil, err
	}
	return &progress, nil
}

func (r *progressRepository) GetProgressByProblemAndUser(problemID string, userID string) (*models.Progress, error) {
	var progress models.Progress
	if err := r.db.Where("problem_id = ? AND user_id = ?", problemID, userID).First(&progress).Error; err != nil {
		log.Printf("Error retrieving progress by problem ID and user ID: %v", err)
		return nil, err
	}
	return &progress, nil
}

func (r *progressRepository) GetProgressWithSubmissionsByProblemAndUser(problemID string, userID string) (*models.Progress, error) {
	var progress models.Progress
	if err := r.db.Preload("Submissions").Preload("Submissions.TestCaseResults").Where("problem_id = ? AND user_id = ?", problemID, userID).First(&progress).Error; err != nil {
		log.Printf("Error retrieving progress with submissions by problem ID and user ID: %v", err)
		return nil, err
	}
	return &progress, nil
}

func (r *progressRepository) GetProgressesWithSubmissionsByUserID(userID string) ([]models.Progress, error) {
	var progresses []models.Progress
	if err := r.db.Preload("Submissions").Where("user_id = ?", userID).Find(&progresses).Error; err != nil {
		log.Printf("Error retrieving progresses with submissions by user ID: %v", err)
		return nil, err
	}
	return progresses, nil
}

func (r *progressRepository) SaveProgress(progress *models.Progress) error {
	if err := r.db.Save(progress).Error; err != nil {
		log.Printf("Error saving progress: %v", err)
		return err
	}
	return nil
}

func (r *progressRepository) DeleteProgress(progress *models.Progress) error {
	if err := r.db.Delete(progress).Error; err != nil {
		log.Printf("Error deleting progress: %v", err)
		return err
	}
	return nil
}

func (r *progressRepository) UpdateBestScore(progressID string, newScore int) error {
	if err := r.db.Model(&models.Progress{}).Where("id = ? AND best_score < ?", progressID, newScore).Update("best_score", newScore).Error; err != nil {
		log.Printf("Error updating best score for progress ID %s: %v", progressID, err)
		return err
	}
	return nil
}
