package controller

import (
	"net/http"
	"strings"

	"ddp0_grader/app/repository"

	"github.com/gin-gonic/gin"
)

type LeaderboardController struct {
	leaderboardRepo repository.LeaderboardRepository
}

func NewLeaderboardController(leaderboardRepo repository.LeaderboardRepository) *LeaderboardController {
	return &LeaderboardController{
		leaderboardRepo: leaderboardRepo,
	}
}

func (lc *LeaderboardController) RegisterRoutes(router gin.IRouter) {
	leaderboard := router.Group("/leaderboard")
	{
		leaderboard.GET("/problem/:problemID", lc.getLeaderboardByProblemID)
		leaderboard.GET("/all-time", lc.getAllTimeLeaderboard)
	}
}

func (lc *LeaderboardController) getLeaderboardByProblemID(c *gin.Context) {
	problemID := strings.TrimSpace(c.Param("problemID"))
	if problemID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "problem id is required"})
		return
	}
	leaderboard, err := lc.leaderboardRepo.GetPublicLeaderboardByProblemID(problemID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve leaderboard"})
		return
	}
	c.JSON(http.StatusOK, leaderboard)
}

func (lc *LeaderboardController) getAllTimeLeaderboard(c *gin.Context) {
	leaderboard, err := lc.leaderboardRepo.GetAllTimeLeaderboard()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve leaderboard"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"scores": leaderboard})
}
