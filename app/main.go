package app

import (
	"ddp0_grader/app/config"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	config.InitConfig()

	r := gin.Default()
	allowed_origins := config.GetEnv("ALLOWED_ORIGINS")

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{allowed_origins},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	r.Run(":" + config.GetEnv("PORT"))
}
