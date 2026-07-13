package main

import (
	"log"

	"ddp0_grader/app/config"
	"ddp0_grader/app/controller"
	"ddp0_grader/app/repository"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	config.InitConfig()

	problemRepo := repository.NewProblemRepository(config.DB)
	submissionRepo := repository.NewSubmissionRepository(config.DB)
	resultRepo := repository.NewTestCaseResultRepository(config.DB)
	submissionController, err := controller.NewSubmissionController(problemRepo, submissionRepo, resultRepo)
	if err != nil {
		log.Fatal(err)
	}

	router := gin.Default()
	router.MaxMultipartMemory = 2 << 20
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{config.GetEnv("ALLOWED_ORIGINS")},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	controller.NewHealthController().RegisterRoutes(router)
	submissionController.RegisterRoutes(router)

	if err := router.Run(":" + config.GetEnv("PORT")); err != nil {
		log.Fatal(err)
	}
}
