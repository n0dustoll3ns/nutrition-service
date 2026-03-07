package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yourusername/auth-service/internal/model"
	"github.com/yourusername/auth-service/internal/repository"
)

// FoodHandler handles food-related HTTP requests
type FoodHandler struct {
	foodRepo   repository.FoodRepository
	recipeRepo repository.RecipeRepository
}

// NewFoodHandler creates a new FoodHandler
func NewFoodHandler(foodRepo repository.FoodRepository, recipeRepo repository.RecipeRepository) *FoodHandler {
	return &FoodHandler{
		foodRepo:   foodRepo,
		recipeRepo: recipeRepo,
	}
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

// SearchCombined handles GET /api/v1/search/combined
// @Summary Combined search for foods and recipes
// @Description Search for both foods and recipes with pagination
// @Tags search
// @Accept json
// @Produce json
// @Param q query string true "Search query"
// @Param limit query int false "Number of results per page (default: 20)" default(20)
// @Param offset query int false "Offset for pagination (default: 0)" default(0)
// @Param include_foods query bool false "Include foods in search (default: true)" default(true)
// @Param include_recipes query bool false "Include recipes in search (default: true)" default(true)
// @Param recipes_public_only query bool false "Only include public recipes (default: true)" default(true)
// @Success 200 {object} model.CombinedSearchResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/search/combined [get]
func (h *FoodHandler) SearchCombined(c *gin.Context) {
	var req struct {
		Query            string `form:"q" binding:"required"`
		Limit            int    `form:"limit,default=20"`
		Offset           int    `form:"offset,default=0"`
		IncludeFoods     bool   `form:"include_foods,default=true"`
		IncludeRecipes   bool   `form:"include_recipes,default=true"`
		RecipesPublicOnly bool  `form:"recipes_public_only,default=true"`
	}
	
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

	// Calculate limits for each type
	foodLimit := 0
	recipeLimit := 0
	
	if req.IncludeFoods && req.IncludeRecipes {
		// Split limit between foods and recipes
		foodLimit = req.Limit / 2
		recipeLimit = req.Limit - foodLimit
	} else if req.IncludeFoods {
		foodLimit = req.Limit
	} else if req.IncludeRecipes {
		recipeLimit = req.Limit
	} else {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Message: "At least one of include_foods or include_recipes must be true",
		})
		return
	}

	// Search foods
	var foods []*model.FoodWithNutrients
	var foodTotal int
	var foodErr error
	
	if req.IncludeFoods && foodLimit > 0 {
		foods, foodTotal, foodErr = h.foodRepo.SearchFoods(c.Request.Context(), req.Query, foodLimit, req.Offset)
		if foodErr != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error:   "Internal server error",
				Message: foodErr.Error(),
			})
			return
		}
	}

	// Search recipes
	var recipes []*model.RecipeWithIngredients
	var recipeTotal int
	var recipeErr error
	
	if req.IncludeRecipes && recipeLimit > 0 {
		recipes, recipeTotal, recipeErr = h.recipeRepo.SearchRecipes(c.Request.Context(), req.Query, req.RecipesPublicOnly, recipeLimit, req.Offset)
		if recipeErr != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error:   "Internal server error",
				Message: recipeErr.Error(),
			})
			return
		}
	}

	// Combine results
	combinedResults := make([]*model.CombinedSearchResult, 0, len(foods)+len(recipes))
	
	// Add foods
	for _, food := range foods {
		combinedResults = append(combinedResults, &model.CombinedSearchResult{
			Type: "food",
			Food: food,
		})
	}
	
	// Add recipes
	for _, recipe := range recipes {
		combinedResults = append(combinedResults, &model.CombinedSearchResult{
			Type:   "recipe",
			Recipe: recipe,
		})
	}

	// Calculate total
	total := foodTotal + recipeTotal

	// Calculate pagination
	totalPages := 0
	if total > 0 {
		totalPages = (total + req.Limit - 1) / req.Limit
	}
	page := (req.Offset / req.Limit) + 1

	response := model.CombinedSearchResponse{
		Data: combinedResults,
		Pagination: &model.Pagination{
			Page:       page,
			Limit:      req.Limit,
			Total:      total,
			TotalPages: totalPages,
		},
	}

	c.JSON(http.StatusOK, response)
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}
