package controller

import (
	"context"
	"errors"
	"log"
	"net/http"

	"ddp0_grader/app/models"
	"ddp0_grader/app/storage"
	"ddp0_grader/app/usecase/problem"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const maxProblemRequestBytes = 96 << 10

type problemRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Author      string `json:"created_by"`
	Tag         string `json:"tag"`
	Difficulty  string `json:"difficulty"`
	TimeLimit   int    `json:"time_limit"`
	MemoryLimit int    `json:"memory_limit"`
	Hint        string `json:"hint"`
}

type thumbnailUpdater interface {
	SetThumbnailKey(ctx context.Context, id, thumbnailKey string) (models.Problem, error)
}

type ProblemController struct {
	useCase    problem.UseCase
	thumbnails storage.ThumbnailStorage
}

func NewProblemController(useCase problem.UseCase, thumbnails ...storage.ThumbnailStorage) *ProblemController {
	var thumbnailStorage storage.ThumbnailStorage
	if len(thumbnails) > 0 {
		thumbnailStorage = thumbnails[0]
	}
	return &ProblemController{useCase: useCase, thumbnails: thumbnailStorage}
}

func (controller *ProblemController) RegisterRoutes(router gin.IRouter) {
	controller.RegisterReadRoutes(router)
	controller.RegisterWriteRoutes(router)
}

func (controller *ProblemController) RegisterReadRoutes(router gin.IRouter) {
	problems := router.Group("/problems")
	{
		problems.GET("", controller.getAll)
		problems.GET("/:id", controller.getByID)
	}
}

func (controller *ProblemController) RegisterWriteRoutes(router gin.IRouter) {
	problems := router.Group("/problems")
	{
		problems.POST("", controller.create)
		problems.PATCH("/:id", controller.update)
		problems.DELETE("/:id", controller.delete)
		problems.POST("/:id/thumbnail", controller.uploadThumbnail)
		problems.DELETE("/:id/thumbnail", controller.deleteThumbnail)
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
	c.JSON(http.StatusCreated, controller.response(c, created))
}

func (controller *ProblemController) getAll(c *gin.Context) {
	problems, err := controller.useCase.GetAll(c.Request.Context())
	if err != nil {
		writeProblemError(c, err)
		return
	}
	// Avoid generating signed thumbnail URLs for every catalog entry. The user
	// receives a URL only after opening one problem.
	c.JSON(http.StatusOK, problems)
}

func (controller *ProblemController) getByID(c *gin.Context) {
	problem, err := controller.useCase.GetByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeProblemError(c, err)
		return
	}
	c.JSON(http.StatusOK, controller.response(c, problem))
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
	c.JSON(http.StatusOK, controller.response(c, updated))
}

func (controller *ProblemController) delete(c *gin.Context) {
	item, err := controller.useCase.GetByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeProblemError(c, err)
		return
	}
	if err := controller.useCase.Delete(c.Request.Context(), c.Param("id")); err != nil {
		writeProblemError(c, err)
		return
	}
	if controller.thumbnails != nil && item.ThumbnailKey != "" {
		if err := controller.thumbnails.Delete(c.Request.Context(), item.ThumbnailKey); err != nil {
			log.Printf("delete thumbnail %s for removed problem: %v", item.ThumbnailKey, err)
		}
	}
	c.Status(http.StatusNoContent)
}

func (controller *ProblemController) uploadThumbnail(c *gin.Context) {
	if controller.thumbnails == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "thumbnail storage is not configured"})
		return
	}
	updater, ok := controller.useCase.(thumbnailUpdater)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "thumbnail updates are unavailable"})
		return
	}
	current, err := controller.useCase.GetByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeProblemError(c, err)
		return
	}
	content, contentType, err := readThumbnail(c)
	if err != nil {
		writeThumbnailError(c, err)
		return
	}
	key, err := controller.thumbnails.Upload(c.Request.Context(), current.ID, content, contentType)
	if err != nil {
		log.Printf("upload thumbnail for problem %s: %v", current.ID, err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to store thumbnail"})
		return
	}
	updated, err := updater.SetThumbnailKey(c.Request.Context(), current.ID, key)
	if err != nil {
		if cleanupErr := controller.thumbnails.Delete(c.Request.Context(), key); cleanupErr != nil {
			log.Printf("clean up failed thumbnail %s: %v", key, cleanupErr)
		}
		writeProblemError(c, err)
		return
	}
	if current.ThumbnailKey != "" && current.ThumbnailKey != key {
		if err := controller.thumbnails.Delete(c.Request.Context(), current.ThumbnailKey); err != nil {
			log.Printf("delete replaced thumbnail %s: %v", current.ThumbnailKey, err)
		}
	}
	c.JSON(http.StatusOK, controller.response(c, updated))
}

func (controller *ProblemController) deleteThumbnail(c *gin.Context) {
	if controller.thumbnails == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "thumbnail storage is not configured"})
		return
	}
	updater, ok := controller.useCase.(thumbnailUpdater)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "thumbnail updates are unavailable"})
		return
	}
	current, err := controller.useCase.GetByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeProblemError(c, err)
		return
	}
	if current.ThumbnailKey == "" {
		c.Status(http.StatusNoContent)
		return
	}
	if _, err := updater.SetThumbnailKey(c.Request.Context(), current.ID, ""); err != nil {
		writeProblemError(c, err)
		return
	}
	if err := controller.thumbnails.Delete(c.Request.Context(), current.ThumbnailKey); err != nil {
		log.Printf("delete thumbnail %s: %v", current.ThumbnailKey, err)
	}
	c.Status(http.StatusNoContent)
}

func bindProblemRequest(c *gin.Context) (problemRequest, bool) {
	var request problemRequest
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxProblemRequestBytes)
	if err := c.ShouldBindJSON(&request); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "problem request is too large"})
			return problemRequest{}, false
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON request"})
		return problemRequest{}, false
	}
	return request, true
}

func (request problemRequest) toInput() problem.CreateInput {
	return problem.CreateInput{Title: request.Title, Description: request.Description, Author: request.Author, Tag: request.Tag, Difficulty: request.Difficulty, TimeLimit: request.TimeLimit, MemoryLimit: request.MemoryLimit, Hint: request.Hint}
}

func (controller *ProblemController) response(c *gin.Context, item models.Problem) models.Problem {
	if controller.thumbnails == nil || item.ThumbnailKey == "" {
		return item
	}
	url, err := controller.thumbnails.PresignedURL(c.Request.Context(), item.ThumbnailKey)
	if err != nil {
		log.Printf("create thumbnail URL for problem %s: %v", item.ID, err)
		return item
	}
	item.ThumbnailURL = url
	return item
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
