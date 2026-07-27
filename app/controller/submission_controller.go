package controller

import (
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"ddp0_grader/app/config"
	"ddp0_grader/app/usecase/grading"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const maxSourceSize = 1 << 20

type SubmissionController struct {
	grading      grading.UseCase
	gradeLimiter GradeRateLimiter
}

func NewSubmissionController(grading grading.UseCase, limiters ...GradeRateLimiter) *SubmissionController {
	controller := &SubmissionController{grading: grading}
	if len(limiters) > 0 {
		controller.gradeLimiter = limiters[0]
	}
	return controller
}

func (controller *SubmissionController) RegisterRoutes(router gin.IRouter) {
	submission := router.Group("/submissions")
	{
		submission.POST("/grade", controller.limitGrade, controller.grade)
		submission.GET("/:id", controller.getByID)
	}
	// Quiz submissions originate from the backend's Celery worker and are
	// idempotent. Do not rate-limit them: a 429 would only delay recovery from
	// a response that was lost after the grader already accepted the job.
	router.POST("/quiz/submissions/grade", controller.gradeQuiz)
}

func (controller *SubmissionController) limitGrade(c *gin.Context) {
	if controller.gradeLimiter == nil {
		return
	}
	userID, ok := config.AuthenticatedUserID(c)
	if !ok {
		return
	}
	allowed, retryAfter, err := controller.gradeLimiter.Allow(c.Request.Context(), userID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "grade rate limiter unavailable"})
		return
	}
	if allowed {
		return
	}
	retrySeconds := int(retryAfter / time.Second)
	if retryAfter%time.Second != 0 {
		retrySeconds++
	}
	if retrySeconds < 1 {
		retrySeconds = 1
	}
	c.Header("Retry-After", strconv.Itoa(retrySeconds))
	c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "grade request rate limit exceeded"})
}

func (controller *SubmissionController) getByID(c *gin.Context) {
	userID, ok := config.AuthenticatedUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authenticated user"})
		return
	}
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "submission id is required"})
		return
	}
	submission, err := controller.grading.GetSubmission(c.Request.Context(), id)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": "submission not found"})
		return
	}
	if submission.Progress.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "submission does not belong to the authenticated user"})
		return
	}

	c.JSON(http.StatusOK, submission)
}

func (controller *SubmissionController) grade(c *gin.Context) {
	controller.gradeSubmission(c, "")
}

func (controller *SubmissionController) gradeQuiz(c *gin.Context) {
	idempotencyKey := strings.TrimSpace(c.GetHeader("Idempotency-Key"))
	if idempotencyKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Idempotency-Key header is required"})
		return
	}
	controller.gradeSubmission(c, idempotencyKey)
}

func (controller *SubmissionController) gradeSubmission(c *gin.Context, idempotencyKey string) {
	problemID := strings.TrimSpace(c.PostForm("problem_id"))
	userID, ok := config.AuthenticatedUserID(c)
	accessToken, hasAccessToken := config.AuthenticatedAccessToken(c)
	if problemID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "problem_id is required"})
		return
	}
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authenticated user"})
		return
	}
	if !hasAccessToken {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authenticated access token"})
		return
	}

	header, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file field is required"})
		return
	}
	if strings.ToLower(filepath.Ext(header.Filename)) != ".py" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "only .py files are accepted"})
		return
	}
	if header.Size > maxSourceSize {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "source file must be at most 1 MiB"})
		return
	}

	file, err := header.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot open uploaded file"})
		return
	}
	defer file.Close()
	source, err := io.ReadAll(io.LimitReader(file, maxSourceSize+1))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot read uploaded file"})
		return
	}
	if int64(len(source)) > maxSourceSize {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "source file must be at most 1 MiB"})
		return
	}

	submission, err := controller.grading.Submit(c.Request.Context(), grading.SubmitInput{
		ProblemID:      problemID,
		UserID:         userID,
		AccessToken:    accessToken,
		SourceCode:     string(source),
		IdempotencyKey: idempotencyKey,
	})
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
		} else if errors.Is(err, grading.ErrTooManyTestCases) {
			status = http.StatusBadRequest
		} else if errors.Is(err, grading.ErrIdempotencyKeyConflict) {
			status = http.StatusConflict
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{"submission_id": submission.ID, "status": submission.Status})
}
