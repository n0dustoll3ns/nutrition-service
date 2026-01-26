package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yourusername/auth-service/internal/model"
	"github.com/yourusername/auth-service/internal/repository"
)

// FoodHandler handles food-related HTTP requests
type FoodHandler struct {
	foodRepo repository.FoodRepository
}

// NewFoodHandler creates a new FoodHandler
func NewFoodHandler(foodRepo repository.FoodRepository) *FoodHandler {
	return &FoodHandler{foodRepo: foodRepo}
}

// SearchFoods handles GET /api/v1/foods/search
// @Summary Search for foods
// @Description Search for foods by description with pagination
// @Tags foods
// @Accept json
// @Produce json
// @Param q query string true "Search query"
// @Param limit query int false "Number of results per page (default: 20)" default(20)
// @Param offset query int false "Offset for pagination (default: 0)" default(0)
// @Success 200 {object} model.SearchFoodResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/foods/search [get]
func (h *FoodHandler) SearchFoods(c *gin.Context) {
	var req model.SearchFoodRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request parameters",
			Message: err.Error(),
		})
		return
	}

	// Validate parameters
	if req.Query == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Message: "Search query is required",
		})
		return
	}

	if req.Limit <= 0 {
		req.Limit = 20
	}
	if req.Limit > 100 {
		req.Limit = 100
	}
	if req.Offset < 0 {
		req.Offset = 0
	}

	// Search foods
	foods, total, err := h.foodRepo.SearchFoods(c.Request.Context(), req.Query, req.Limit, req.Offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Message: err.Error(),
		})
		return
	}

	// Calculate pagination
	totalPages := 0
	if total > 0 {
		totalPages = (total + req.Limit - 1) / req.Limit
	}
	page := (req.Offset / req.Limit) + 1

	response := model.SearchFoodResponse{
		Data: foods,
		Pagination: &model.Pagination{
			Page:       page,
			Limit:      req.Limit,
			Total:      total,
			TotalPages: totalPages,
		},
	}

	c.JSON(http.StatusOK, response)
}

// GetFoodByID handles GET /api/v1/foods/:id
// @Summary Get food by ID
// @Description Get food details by FDC ID
// @Tags foods
// @Accept json
// @Produce json
// @Param id path int true "FDC ID"
// @Success 200 {object} model.FoodWithNutrients
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/foods/{id} [get]
func (h *FoodHandler) GetFoodByID(c *gin.Context) {
	var req struct {
		ID int `uri:"id" binding:"required"`
	}
	if err := c.ShouldBindUri(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid food ID",
			Message: err.Error(),
		})
		return
	}

	food, err := h.foodRepo.GetFoodByID(c.Request.Context(), req.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Message: err.Error(),
		})
		return
	}

	if food == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Food not found",
			Message: "Food with the specified ID does not exist",
		})
		return
	}

	c.JSON(http.StatusOK, food)
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}