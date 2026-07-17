package repository

import (
	"gorm.io/gorm"
)

type LeaderboardEntry struct {
	Email     string `json:"email"`
	BestScore int    `json:"best_score"`
}

type AdminLeaderboardEntry struct {
	Rank                int    `json:"rank"`
	Email               string `json:"email"`
	TotalScore          int    `json:"total_score"`
	SolvedProblems      int    `json:"solved_problems"`
	TotalSubmissions    int    `json:"total_submissions"`
	AcceptedSubmissions int    `json:"accepted_submissions"`
}

type LeaderboardRepository interface {
	GetPublicLeaderboardByProblemID(problemID string) ([]LeaderboardEntry, error)
	GetAllTimeLeaderboard() ([]LeaderboardEntry, error)
	GetAdminLeaderboard() ([]AdminLeaderboardEntry, error)
}

func (r *leaderboardRepository) GetAdminLeaderboard() ([]AdminLeaderboardEntry, error) {
	var leaderboard []AdminLeaderboardEntry
	progressTotals := r.db.Table("progresses").
		Select("user_id, SUM(best_score) AS total_score, SUM(CASE WHEN best_score = ? THEN 1 ELSE 0 END) AS solved_problems", 100).
		Group("user_id")
	submissionTotals := r.db.Table("submissions").
		Select("progresses.user_id, COUNT(submissions.id) AS total_submissions, SUM(CASE WHEN submissions.status = ? THEN 1 ELSE 0 END) AS accepted_submissions", "accepted").
		Joins("JOIN progresses ON submissions.progress_id = progresses.id").
		Group("progresses.user_id")

	err := r.db.Table("(?) AS progress_totals", progressTotals).
		Select("users.email, progress_totals.total_score, progress_totals.solved_problems, COALESCE(submission_totals.total_submissions, 0) AS total_submissions, COALESCE(submission_totals.accepted_submissions, 0) AS accepted_submissions").
		Joins("JOIN users ON progress_totals.user_id = users.id").
		Joins("LEFT JOIN (?) AS submission_totals ON submission_totals.user_id = users.id", submissionTotals).
		Order("progress_totals.total_score DESC").
		Order("users.email ASC").
		Scan(&leaderboard).Error
	if err != nil {
		return nil, err
	}
	for index := range leaderboard {
		leaderboard[index].Rank = index + 1
	}
	return leaderboard, nil
}

type leaderboardRepository struct {
	db *gorm.DB
}

func NewLeaderboardRepository(db *gorm.DB) LeaderboardRepository {
	return &leaderboardRepository{db}
}

func (r *leaderboardRepository) GetPublicLeaderboardByProblemID(problemID string) ([]LeaderboardEntry, error) {
	var leaderboard []LeaderboardEntry
	err := r.db.Table("progresses").
		Select("users.email, progresses.best_score").
		Joins("JOIN users ON progresses.user_id = users.id").
		Where("progresses.problem_id = ?", problemID).
		Order("progresses.best_score DESC").
		Scan(&leaderboard).Error

	if err != nil {
		return nil, err
	}
	return leaderboard, nil
}

func (r *leaderboardRepository) GetAllTimeLeaderboard() ([]LeaderboardEntry, error) {
	var leaderboard []LeaderboardEntry
	err := r.db.Table("progresses").
		Select("users.email, SUM(progresses.best_score) as best_score").
		Joins("JOIN users ON progresses.user_id = users.id").
		Group("users.email").
		Order("best_score DESC").
		Scan(&leaderboard).Error

	if err != nil {
		return nil, err
	}
	return leaderboard, nil
}
