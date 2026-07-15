package controller

import (
	"errors"
	"net/http"

	"ddp0_grader/app/config"
	"ddp0_grader/app/usecase/progress"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ProgressController struct {
	useCase progress.UseCase
}

func NewProgressController(useCase progress.UseCase) *ProgressController {
	return &ProgressController{useCase: useCase}
}

func (controller *ProgressController) RegisterRoutes(router gin.IRouter) {
	progresses := router.Group("/progresses")
	{
		progresses.GET("", controller.getByUserID)
		progresses.GET("/:problemID", controller.getByUserAndProblemID)
	}
}

func (controller *ProgressController) getByUserID(c *gin.Context) {
	userID, ok := config.AuthenticatedUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authenticated user"})
		return
	}
	progresses, err := controller.useCase.GetByUserID(c.Request.Context(), userID)
	if err != nil {
		writeProgressError(c, err)
		return
	}
	c.JSON(http.StatusOK, progresses)
}

func (controller *ProgressController) getByUserAndProblemID(c *gin.Context) {
	userID, ok := config.AuthenticatedUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authenticated user"})
		return
	}
	progress, err := controller.useCase.GetByUserAndProblemID(c.Request.Context(), userID, c.Param("problemID"))
	if err != nil {
		writeProgressError(c, err)
		return
	}
	c.JSON(http.StatusOK, progress)
}

func writeProgressError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, progress.ErrInvalidInput):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, gorm.ErrRecordNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "progress not found"})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}
}
