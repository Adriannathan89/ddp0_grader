package controller

import (
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"ddp0_grader/app/config"
	"ddp0_grader/app/repository"
	"ddp0_grader/app/usecase/grading"
	problemUseCase "ddp0_grader/app/usecase/problem"
	submissionUseCase "ddp0_grader/app/usecase/submission"
	resultUseCase "ddp0_grader/app/usecase/testcaseresult"
	"ddp0_grader/pkg/queue"
	"ddp0_grader/pkg/runner"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const maxSourceSize = 1 << 20

type SubmissionController struct {
	grading grading.UseCase
}

func NewSubmissionController(
	problemRepo repository.ProblemRepository,
	submissionRepo repository.SubmissionRepository,
	resultRepo repository.TestCaseResultRepository,
) (*SubmissionController, error) {
	jobQueue, err := queue.NewWithClient(config.RedisClient, queue.Config{
		Stream:   "grader:jobs",
		Group:    "grader-workers",
		Consumer: "api-worker",
	})
	if err != nil {
		return nil, err
	}

	grader := runner.New(runner.Config{
		Image:           "python:3.12-slim",
		OutputLimit:     1 << 20,
		DefaultTime:     2 * time.Second,
		DefaultMemoryMB: 256,
	})
	gradingUseCase := grading.NewUseCase(
		problemUseCase.NewGetProblemUseCase(problemRepo),
		submissionUseCase.NewCreateSubmissionUseCase(submissionRepo),
		submissionUseCase.NewUpdateSubmissionUseCase(submissionRepo),
		resultUseCase.NewBatchCreateTestCaseResultUseCase(resultRepo),
		jobQueue,
		grader,
	)

	controller := &SubmissionController{grading: gradingUseCase}
	go controller.startWorker(jobQueue)
	return controller, nil
}

func (controller *SubmissionController) startWorker(jobQueue *queue.Queue) {
	if err := jobQueue.WorkN(context.Background(), 10, controller.grading.GradeJob); err != nil && !errors.Is(err, context.Canceled) {
		log.Printf("grader workers stopped: %v", err)
	}
}

func (controller *SubmissionController) RegisterRoutes(router *gin.Engine) {
	router.POST("/submissions/grade", controller.grade)
}

func (controller *SubmissionController) grade(c *gin.Context) {
	problemID := strings.TrimSpace(c.PostForm("problem_id"))
	userID := strings.TrimSpace(c.PostForm("user_id"))
	if problemID == "" || userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "problem_id and user_id are required"})
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
