package controller

import (
	"net/http"
	"strconv"
	"strings"

	"ddp0_grader/app/repository"

	"github.com/gin-gonic/gin"
)

type AdminReviewController struct {
	submissions repository.SubmissionRepository
}

func NewAdminReviewController(submissions repository.SubmissionRepository) *AdminReviewController {
	return &AdminReviewController{submissions: submissions}
}

func (controller *AdminReviewController) RegisterRoutes(router gin.IRouter) {
	router.GET("/admin/submissions", controller.list)
}

func (controller *AdminReviewController) list(c *gin.Context) {
	limit := boundedInt(c.Query("limit"), 25, 1, 100)
	offset := boundedInt(c.Query("offset"), 0, 0, 1_000_000)
	items, total, err := controller.submissions.GetAdminSubmissions(repository.AdminSubmissionFilter{
		ProblemID: c.Query("problem_id"),
		UserQuery: c.Query("user"),
		Status:    c.Query("status"),
		Limit:     limit,
		Offset:    offset,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve submissions"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"result": items, "total": total, "limit": limit, "offset": offset})
}

func boundedInt(value string, fallback, minimum, maximum int) int {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || parsed < minimum {
		return fallback
	}
	if parsed > maximum {
		return maximum
	}
	return parsed
}
