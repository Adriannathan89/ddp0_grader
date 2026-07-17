package repository

import (
	"ddp0_grader/app/models"
	"log"
	"strings"
	"time"

	"gorm.io/gorm"
)

type SubmissionRepository interface {
	GetSubmissionByID(id string) (*models.Submission, error)
	GetSubmissionByIDWithPreloaded(id string) (*models.Submission, error)
	SaveSubmission(submission *models.Submission) error
	DeleteSubmission(submission *models.Submission) error
	GetAdminSubmissions(filter AdminSubmissionFilter) ([]AdminSubmission, int64, error)
}

type AdminSubmissionFilter struct {
	ProblemID string
	UserQuery string
	Status    string
	Limit     int
	Offset    int
}

type AdminSubmission struct {
	ID           string    `json:"id"`
	Status       string    `json:"status"`
	Score        int       `json:"score"`
	TotalRunTime int       `json:"total_run_time"`
	SourceCode   string    `json:"source_code"`
	CreatedAt    time.Time `json:"created_at"`
	UserID       string    `json:"user_id"`
	UserEmail    string    `json:"user_email"`
	ProblemID    string    `json:"problem_id"`
	ProblemTitle string    `json:"problem_title"`
}

type submissionRepository struct {
	db *gorm.DB
}

func NewSubmissionRepository(db *gorm.DB) SubmissionRepository {
	return &submissionRepository{db}
}

func (r *submissionRepository) GetSubmissionByID(id string) (*models.Submission, error) {
	var submission models.Submission
	if err := r.db.First(&submission, "id = ?", id).Error; err != nil {
		log.Printf("Error retrieving submission by ID: %v", err)
		return nil, err
	}
	return &submission, nil
}

func (r *submissionRepository) GetSubmissionByIDWithPreloaded(id string) (*models.Submission, error) {
	var submission models.Submission
	if err := r.db.Preload("Progress").Preload("TestCaseResults").First(&submission, "id = ?", id).Error; err != nil {
		log.Printf("Error retrieving submission by ID with preloaded test case results: %v", err)
		return nil, err
	}
	return &submission, nil
}

func (r *submissionRepository) SaveSubmission(submission *models.Submission) error {
	if err := r.db.Save(submission).Error; err != nil {
		log.Printf("Error saving submission: %v", err)
		return err
	}
	return nil
}

func (r *submissionRepository) DeleteSubmission(submission *models.Submission) error {
	if err := r.db.Delete(submission).Error; err != nil {
		log.Printf("Error deleting submission: %v", err)
		return err
	}
	return nil
}

func (r *submissionRepository) GetAdminSubmissions(filter AdminSubmissionFilter) ([]AdminSubmission, int64, error) {
	query := r.db.Table("submissions").
		Joins("JOIN progresses ON submissions.progress_id = progresses.id").
		Joins("JOIN users ON progresses.user_id = users.id").
		Joins("JOIN problems ON progresses.problem_id = problems.id")
	if value := strings.TrimSpace(filter.ProblemID); value != "" {
		query = query.Where("progresses.problem_id = ?", value)
	}
	if value := strings.TrimSpace(filter.Status); value != "" {
		query = query.Where("submissions.status = ?", value)
	}
	if value := strings.TrimSpace(filter.UserQuery); value != "" {
		query = query.Where("users.email ILIKE ?", "%"+value+"%")
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var submissions []AdminSubmission
	err := query.Select(`submissions.id, submissions.status, submissions.score, submissions.total_run_time,
		submissions.source_code, submissions.created_at, progresses.user_id, users.email AS user_email,
		progresses.problem_id, problems.title AS problem_title`).
		Order("submissions.created_at DESC").
		Limit(filter.Limit).
		Offset(filter.Offset).
		Scan(&submissions).Error
	return submissions, total, err
}
