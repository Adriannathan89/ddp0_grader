package controller

import (
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"ddp0_grader/app/config"
	"ddp0_grader/app/usecase/grading"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const maxSourceSize = 1 << 20

type SubmissionController struct {
	grading grading.UseCase
}

func NewSubmissionController(grading grading.UseCase) *SubmissionController {
	return &SubmissionController{grading: grading}
}

func (controller *SubmissionController) RegisterRoutes(router gin.IRouter) {
	submission := router.Group("/submissions")
	{
		submission.POST("/grade", controller.grade)
		submission.GET("/:id", controller.getByID)
	}
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
	problemID := strings.TrimSpace(c.PostForm("problem_id"))
	userID, ok := config.AuthenticatedUserID(c)
	if problemID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "problem_id is required"})
		return
	}
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authenticated user"})
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
		ProblemID:  problemID,
		UserID:     userID,
		SourceCode: string(source),
	})
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{"submission_id": submission.ID, "status": submission.Status})
}
