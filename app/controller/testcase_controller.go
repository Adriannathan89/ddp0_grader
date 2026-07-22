package controller

import (
	"errors"
	"net/http"

	"ddp0_grader/app/usecase/testcase"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type TestCaseController struct {
	useCase testcase.UseCase
}

const maxTestCaseRequestBytes = testcase.MaxInputBytes + testcase.MaxOutputBytes + 16<<10

type testCaseRequest struct {
	Input    string `json:"input"`
	Output   string `json:"output"`
	IsHidden bool   `json:"is_hidden"`
}

func NewTestCaseController(useCase testcase.UseCase) *TestCaseController {
	return &TestCaseController{useCase: useCase}
}

func (controller *TestCaseController) RegisterRoutes(router gin.IRouter) {
	controller.RegisterReadRoutes(router)
	controller.RegisterWriteRoutes(router)
}

func (controller *TestCaseController) RegisterReadRoutes(router gin.IRouter) {
	problems := router.Group("/problems/:id/testcases")
	{
		problems.GET("", controller.getByProblemID)
	}
	testCases := router.Group("/testcases")
	{
		testCases.GET("/:id", controller.getByID)
	}
}

func (controller *TestCaseController) RegisterWriteRoutes(router gin.IRouter) {
	problems := router.Group("/problems/:id/testcases")
	{
		problems.POST("", controller.create)
	}
	testCases := router.Group("/testcases")
	{
		testCases.PATCH("/:id", controller.update)
		testCases.DELETE("/:id", controller.delete)
	}
}

// RegisterAdminRoutes exposes testcase content, including hidden input and
// output. The caller must mount this only behind the Django-admin middleware.
func (controller *TestCaseController) RegisterAdminRoutes(router gin.IRouter) {
	adminProblems := router.Group("/admin/problems/:id/testcases")
	{
		adminProblems.GET("", controller.getAllByProblemID)
	}
}

func (controller *TestCaseController) create(c *gin.Context) {
	request, ok := bindTestCaseRequest(c)
	if !ok {
		return
	}
	created, err := controller.useCase.Create(c.Request.Context(), testcase.CreateInput{
		ProblemID: c.Param("id"), Input: request.Input, Output: request.Output, IsHidden: request.IsHidden,
	})
	if err != nil {
		writeTestCaseError(c, err)
		return
	}
	c.JSON(http.StatusCreated, created)
}

func (controller *TestCaseController) getByProblemID(c *gin.Context) {
	testCases, err := controller.useCase.GetByProblemID(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeTestCaseError(c, err)
		return
	}
	c.JSON(http.StatusOK, testCases)
}

func (controller *TestCaseController) getAllByProblemID(c *gin.Context) {
	testCases, err := controller.useCase.GetAllByProblemID(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeTestCaseError(c, err)
		return
	}
	c.JSON(http.StatusOK, testCases)
}

func (controller *TestCaseController) getByID(c *gin.Context) {
	testCase, err := controller.useCase.GetByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeTestCaseError(c, err)
		return
	}
	c.JSON(http.StatusOK, testCase)
}

func (controller *TestCaseController) update(c *gin.Context) {
	request, ok := bindTestCaseRequest(c)
	if !ok {
		return
	}
	updated, err := controller.useCase.Update(c.Request.Context(), c.Param("id"), testcase.UpdateInput{
		Input: request.Input, Output: request.Output, IsHidden: request.IsHidden,
	})
	if err != nil {
		writeTestCaseError(c, err)
		return
	}
	c.JSON(http.StatusOK, updated)
}

func (controller *TestCaseController) delete(c *gin.Context) {
	if err := controller.useCase.Delete(c.Request.Context(), c.Param("id")); err != nil {
		writeTestCaseError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func bindTestCaseRequest(c *gin.Context) (testCaseRequest, bool) {
	var request testCaseRequest
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxTestCaseRequestBytes)
	if err := c.ShouldBindJSON(&request); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "testcase request is too large"})
			return testCaseRequest{}, false
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON request"})
		return testCaseRequest{}, false
	}
	return request, true
}

func writeTestCaseError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, testcase.ErrInvalidInput):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, gorm.ErrRecordNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "resource not found"})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}
}
