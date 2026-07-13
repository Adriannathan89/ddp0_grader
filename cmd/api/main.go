package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"ddp0_grader/app/config"
	"ddp0_grader/app/models"
	"ddp0_grader/app/repository"
	"ddp0_grader/pkg/queue"
	"ddp0_grader/pkg/runner"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const maxSourceSize = 1 << 20 // 1 MiB

func main() {
	config.InitConfig()

	problemRepo := repository.NewProblemRepository(config.DB)
	submissionRepo := repository.NewSubmissionRepository(config.DB)
	resultRepo := repository.NewTestCaseResultRepository(config.DB)
	grader := runner.New(runner.Config{
		Image:           "python:3.12-slim",
		OutputLimit:     1 << 20,
		DefaultTime:     2 * time.Second,
		DefaultMemoryMB: 256,
	})
	jobQueue, err := queue.NewWithClient(config.RedisClient, queue.Config{
		Stream:   "grader:jobs",
		Group:    "grader-workers",
		Consumer: "api-worker",
	})
	if err != nil {
		log.Fatal(err)
	}

	workerCtx := context.Background()
	go func() {
		err := jobQueue.WorkN(workerCtx, 10, func(ctx context.Context, job queue.Job) error {
			return gradeJob(ctx, job, grader, submissionRepo, resultRepo)
		})
		if err != nil && !errors.Is(err, context.Canceled) {
			log.Printf("grader workers stopped: %v", err)
		}
	}()

	r := gin.Default()
	r.MaxMultipartMemory = 2 << 20
	allowedOrigins := config.GetEnv("ALLOWED_ORIGINS")
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{allowedOrigins},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	r.POST("/submissions/grade", submitGrade(problemRepo, submissionRepo, jobQueue))

	if err := r.Run(":" + config.GetEnv("PORT")); err != nil {
		log.Fatal(err)
	}
}

func submitGrade(problemRepo repository.ProblemRepository, submissionRepo repository.SubmissionRepository, jobQueue *queue.Queue) gin.HandlerFunc {
	return func(c *gin.Context) {
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

		problem, err := problemRepo.GetProblemByIDWithPreloaded(problemID)
		if err != nil {
			status := http.StatusInternalServerError
			if errors.Is(err, gorm.ErrRecordNotFound) {
				status = http.StatusNotFound
			}
			c.JSON(status, gin.H{"error": "problem not found"})
			return
		}
		submission := &models.Submission{
			ID:         newID(),
			ProblemID:  problem.ID,
			UserID:     userID,
			SourceCode: string(source),
			Status:     "queued",
		}
		if err := submissionRepo.SaveSubmission(submission); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot save submission"})
			return
		}

		_, err = jobQueue.Enqueue(c.Request.Context(), queue.Job{
			ID:         submission.ID,
			Submission: *submission,
			Problem:    *problem,
			TestCases:  problem.TestCases,
		})
		if err != nil {
			message := "cannot enqueue grading job"
			submission.Status = "queue_error"
			submission.ErrorMessage = &message
			_ = submissionRepo.SaveSubmission(submission)
			c.JSON(http.StatusInternalServerError, gin.H{"error": message})
			return
		}

		c.JSON(http.StatusAccepted, gin.H{"submission_id": submission.ID, "status": submission.Status})
	}
}

func gradeJob(ctx context.Context, job queue.Job, grader *runner.Runner, submissionRepo repository.SubmissionRepository, resultRepo repository.TestCaseResultRepositoryInterface) error {
	results, err := grader.Run(ctx, &job.Submission, &job.Problem, job.TestCases)
	if err != nil {
		message := err.Error()
		job.Submission.Status = "system_error"
		job.Submission.ErrorMessage = &message
		return submissionRepo.SaveSubmission(&job.Submission)
	}

	passed := 0
	totalRuntime := time.Duration(0)
	dbResults := make([]models.TestCaseResult, 0, len(results))
	status := "accepted"
	for _, result := range results {
		if result.Passed {
			passed++
		} else if status == "accepted" {
			status = result.Verdict
		}
		totalRuntime += result.RunTime
		dbResult := models.TestCaseResult{
			ID:           newID(),
			SubmissionID: job.Submission.ID,
			TestCaseID:   result.TestCaseID,
			IsPassed:     result.Passed,
			RunTime:      int(result.RunTime.Milliseconds()),
		}
		if result.Error != nil {
			message := result.Error.Error()
			dbResult.ErrorMessage = &message
		} else if result.Stderr != "" {
			dbResult.ErrorMessage = &result.Stderr
		}
		dbResults = append(dbResults, dbResult)
	}

	job.Submission.Status = status
	if len(results) > 0 {
		job.Submission.Score = passed * 100 / len(results)
	}
	job.Submission.RunTime = int(totalRuntime.Milliseconds())
	if err := resultRepo.BatchSaveTestCaseResults(dbResults); err != nil {
		return err
	}
	return submissionRepo.SaveSubmission(&job.Submission)
}

func newID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return hex.EncodeToString([]byte(time.Now().Format("20060102150405.000000000")))
	}
	return hex.EncodeToString(b)
}
