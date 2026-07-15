package repository

import (
	"gorm.io/gorm"
)

type LeaderboardEntry struct {
	Email     string `json:"email"`
	BestScore int    `json:"best_score"`
}

type LeaderboardRepository interface {
	GetPublicLeaderboardByProblemID(problemID string) ([]LeaderboardEntry, error)
	GetAllTimeLeaderboard() ([]LeaderboardEntry, error)
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
