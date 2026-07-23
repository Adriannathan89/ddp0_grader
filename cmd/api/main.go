package main

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

	"ddp0_grader/app/config"
	"ddp0_grader/app/controller"
	"ddp0_grader/app/repository"
	"ddp0_grader/app/usecase/grading"
	"ddp0_grader/app/usecase/problem"
	progressuc "ddp0_grader/app/usecase/progress"
	"ddp0_grader/app/usecase/testcase"
	"ddp0_grader/pkg/queue"
	"ddp0_grader/pkg/runner"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

const graderWorkers = 4

func main() {
	config.InitConfig()

	problemRepo := repository.NewProblemRepository(config.DB)
	submissionRepo := repository.NewSubmissionRepository(config.DB)
	resultRepo := repository.NewTestCaseResultRepository(config.DB)
	testCaseRepo := repository.NewTestCaseRepository(config.DB)
	progressRepo := repository.NewProgressRepository(config.DB)
	userRepo := repository.NewUserRepository(config.DB)
	leaderboardRepo := repository.NewLeaderboardRepository(config.DB)
	jobQueue, err := queue.NewWithClient(config.RedisClient, queue.Config{
		Stream:           "grader:jobs",
		Group:            "grader-workers",
		MaxAttempts:      3,
		RetryDelay:       10 * time.Second,
		ClaimIdle:        2 * time.Minute,
		DeadLetterMaxLen: 10_000,
	})
	if err != nil {
		log.Fatal(err)
	}
	authMiddleware, err := config.NewJWTAuthMiddlewareFromEnv()
	if err != nil {
		log.Fatal(err)
	}
	adminMiddleware, err := config.NewDjangoAdminMiddlewareFromEnv()
	if err != nil {
		log.Fatal(err)
	}
	userIdentityProvider, err := config.NewDjangoUserProviderFromEnv()
	if err != nil {
		log.Fatal(err)
	}
	grader := runner.New(runner.Config{
		Image:           "python:3.12-slim",
		OutputLimit:     64 << 10,
		DefaultTime:     time.Second,
		DefaultMemoryMB: 64,
		MaxTime:         time.Second,
		MaxMemoryMB:     64,
	})
	gradingUseCase := grading.NewUseCase(problemRepo, submissionRepo, resultRepo, progressRepo, userRepo, userIdentityProvider, jobQueue, grader)
	problemUseCase := problem.NewUseCase(problemRepo)
	testCaseUseCase := testcase.NewUseCase(problemRepo, testCaseRepo)
	progressUseCase := progressuc.NewUseCase(progressRepo)
	submissionController := controller.NewSubmissionController(gradingUseCase)
	adminReviewController := controller.NewAdminReviewController(submissionRepo)
	problemController := controller.NewProblemController(problemUseCase)
	testCaseController := controller.NewTestCaseController(testCaseUseCase)
	progressController := controller.NewProgressController(progressUseCase)
	leaderboardController := controller.NewLeaderboardController(leaderboardRepo)
	go func() {
		if err := jobQueue.WorkNWithExhaustedHandler(context.Background(), graderWorkers, gradingUseCase.GradeJob, gradingUseCase.MarkJobExhausted); err != nil && !errors.Is(err, context.Canceled) {
			log.Printf("grader workers stopped: %v", err)
		}
	}()

	router := gin.Default()
	router.MaxMultipartMemory = 2 << 20
	router.Use(cors.New(cors.Config{
		AllowOrigins:     allowedOrigins(config.GetEnv("ALLOWED_ORIGINS")),
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	api := router.Group("/api")
	controller.NewHealthController().RegisterRoutes(api)
	// Problem and testcase reads remain available to authenticated gym users;
	// every mutation is guarded by Django's server-side admin validation.
	readAPI := api.Group("")
	readAPI.Use(authMiddleware)
	problemController.RegisterReadRoutes(readAPI)
	testCaseController.RegisterReadRoutes(readAPI)
	adminAPI := api.Group("")
	adminAPI.Use(authMiddleware, adminMiddleware)
	problemController.RegisterWriteRoutes(adminAPI)
	testCaseController.RegisterWriteRoutes(adminAPI)
	testCaseController.RegisterAdminRoutes(adminAPI)
	adminReviewController.RegisterRoutes(adminAPI)
	leaderboardController.RegisterAdminRoutes(adminAPI)
	protectedAPI := api.Group("")
	protectedAPI.Use(authMiddleware)
	submissionController.RegisterRoutes(protectedAPI)
	progressController.RegisterRoutes(protectedAPI)
	leaderboardController.RegisterRoutes(protectedAPI)

	if err := router.Run(":" + config.GetEnv("PORT")); err != nil {
		log.Fatal(err)
	}
}

func allowedOrigins(value string) []string {
	parts := strings.Split(value, ",")
	origins := make([]string, 0, len(parts))
	for _, origin := range parts {
		if origin = strings.TrimSpace(origin); origin != "" {
			origins = append(origins, origin)
		}
	}
	return origins
}
