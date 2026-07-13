package controller

import (
	"errors"
	"net/http"

	"ddp0_grader/app/usecase/problem"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ProblemController struct {
	useCase problem.UseCase
}

type problemRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Author      string `json:"created_by"`
	Tag         string `json:"tag"`
	Difficulty  string `json:"difficulty"`
	TimeLimit   int    `json:"time_limit"`
	MemoryLimit int    `json:"memory_limit"`
}

func NewProblemController(useCase problem.UseCase) *ProblemController {
	return &ProblemController{useCase: useCase}
}

func (controller *ProblemController) RegisterRoutes(router *gin.Engine) {
	problems := router.Group("/problems")
	{
		problems.POST("", controller.create)
		problems.GET("", controller.getAll)
		problems.GET("/:id", controller.getByID)
		problems.PATCH("/:id", controller.update)
		problems.DELETE("/:id", controller.delete)
	}
}

func (controller *ProblemController) create(c *gin.Context) {
	request, ok := bindProblemRequest(c)
	if !ok {
		return
	}
	created, err := controller.useCase.Create(c.Request.Context(), request.toInput())
	if err != nil {
		writeProblemError(c, err)
		return
	}
	c.JSON(http.StatusCreated, created)
}

func (controller *ProblemController) getAll(c *gin.Context) {
	problems, err := controller.useCase.GetAll(c.Request.Context())
	if err != nil {
		writeProblemError(c, err)
		return
	}
	c.JSON(http.StatusOK, problems)
}

func (controller *ProblemController) getByID(c *gin.Context) {
	problem, err := controller.useCase.GetByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeProblemError(c, err)
		return
	}
	c.JSON(http.StatusOK, problem)
}

func (controller *ProblemController) update(c *gin.Context) {
	request, ok := bindProblemRequest(c)
	if !ok {
		return
	}
	updated, err := controller.useCase.Update(c.Request.Context(), c.Param("id"), request.toInput())
	if err != nil {
		writeProblemError(c, err)
		return
	}
	c.JSON(http.StatusOK, updated)
}

func (controller *ProblemController) delete(c *gin.Context) {
	if err := controller.useCase.Delete(c.Request.Context(), c.Param("id")); err != nil {
		writeProblemError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func bindProblemRequest(c *gin.Context) (problemRequest, bool) {
	var request problemRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON request"})
		return problemRequest{}, false
	}
	return request, true
}

func (request problemRequest) toInput() problem.CreateInput {
	return problem.CreateInput{Title: request.Title, Description: request.Description, Author: request.Author, Tag: request.Tag, Difficulty: request.Difficulty, TimeLimit: request.TimeLimit, MemoryLimit: request.MemoryLimit}
}

func writeProblemError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, problem.ErrInvalidInput):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, gorm.ErrRecordNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "problem not found"})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}
}
