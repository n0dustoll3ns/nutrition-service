package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/yourusername/auth-service/internal/model"
	"github.com/yourusername/auth-service/internal/repository"
)

// RecipeHandler handles recipe-related HTTP requests
type RecipeHandler struct {
	recipeRepo repository.RecipeRepository
	foodRepo   repository.FoodRepository
}

// NewRecipeHandler creates a new RecipeHandler
func NewRecipeHandler(recipeRepo repository.RecipeRepository, foodRepo repository.FoodRepository) *RecipeHandler {
	return &RecipeHandler{
		recipeRepo: recipeRepo,
		foodRepo:   foodRepo,
	}
}

// CreateRecipe handles POST /api/v1/recipes
func (h *RecipeHandler) CreateRecipe(c *gin.Context) {
	var req model.RecipeCreate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request body",
			Message: err.Error(),
		})
		return
	}

	// Get user ID from context
	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "Unauthorized",
			Message: err.Error(),
		})
		return
	}

	// Validate ingredients
	for _, ingredient := range req.Ingredients {
		if (ingredient.FDCID == nil && ingredient.CustomFoodName == nil) || 
		   (ingredient.FDCID != nil && ingredient.CustomFoodName != nil) {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "Invalid ingredient",
				Message: "Exactly one of fdc_id or custom_food_name must be provided for each ingredient",
			})
			return
		}
	}

	// Calculate total weight
	totalWeight := 0.0
	for _, ingredient := range req.Ingredients {
		totalWeight += ingredient.AmountGrams
	}

	// Create recipe model
	now := time.Now()
	recipe := &model.Recipe{
		UserID:      userID,
		Name:        req.Name,
		Description: req.Description,
		Category:    req.Category,
		IsPublic:    req.IsPublic,
		TotalWeight: totalWeight,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Create recipe ingredients
	var ingredients []*model.RecipeIngredient
	for _, ingredientReq := range req.Ingredients {
		ingredient := &model.RecipeIngredient{
			ID:              uuid.New(),
			FDCID:           ingredientReq.FDCID,
			CustomFoodName:  ingredientReq.CustomFoodName,
			AmountGrams:     ingredientReq.AmountGrams,
			// Nutrients will be calculated later
		}
		ingredients = append(ingredients, ingredient)
	}

	// Create recipe in repository
	createdRecipe, err := h.recipeRepo.CreateRecipe(c.Request.Context(), recipe, ingredients)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, createdRecipe)
}

// GetRecipe handles GET /api/v1/recipes/{id}
func (h *RecipeHandler) GetRecipe(c *gin.Context) {
	recipeID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid recipe ID",
			Message: "Recipe ID must be a number",
		})
		return
	}

	// Get recipe
	recipe, err := h.recipeRepo.GetRecipeByID(c.Request.Context(), recipeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Message: err.Error(),
		})
		return
	}

	if recipe == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Recipe not found",
			Message: "Recipe with the specified ID does not exist",
		})
		return
	}

	// Check if recipe is public or belongs to user
	userID, err := getUserIDFromContext(c)
	if err == nil {
		// User is authenticated
		if recipe.Recipe.UserID != userID && !recipe.Recipe.IsPublic {
			c.JSON(http.StatusForbidden, ErrorResponse{
				Error:   "Forbidden",
				Message: "You don't have permission to view this recipe",
			})
			return
		}
	} else {
		// User is not authenticated, only show public recipes
		if !recipe.Recipe.IsPublic {
			c.JSON(http.StatusUnauthorized, ErrorResponse{
				Error:   "Unauthorized",
				Message: "Authentication required to view this recipe",
			})
			return
		}
	}

	c.JSON(http.StatusOK, recipe)
}

// UpdateRecipe handles PUT /api/v1/recipes/{id}
func (h *RecipeHandler) UpdateRecipe(c *gin.Context) {
	recipeID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid recipe ID",
			Message: "Recipe ID must be a number",
		})
		return
	}

	var req model.RecipeUpdate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request body",
			Message: err.Error(),
		})
		return
	}

	// Get user ID from context
	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "Unauthorized",
			Message: err.Error(),
		})
		return
	}

	// Check if recipe exists and belongs to user
	existingRecipe, err := h.recipeRepo.GetRecipeByID(c.Request.Context(), recipeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Message: err.Error(),
		})
		return
	}
	
	if existingRecipe == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Recipe not found",
			Message: "Recipe with the specified ID does not exist",
		})
		return
	}
	
	if existingRecipe.Recipe.UserID != userID {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error:   "Forbidden",
			Message: "You don't have permission to update this recipe",
		})
		return
	}

	// Update recipe
	err = h.recipeRepo.UpdateRecipe(c.Request.Context(), recipeID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Message: err.Error(),
		})
		return
	}

	// Get updated recipe
	updatedRecipe, err := h.recipeRepo.GetRecipeByID(c.Request.Context(), recipeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, updatedRecipe)
}

// DeleteRecipe handles DELETE /api/v1/recipes/{id}
func (h *RecipeHandler) DeleteRecipe(c *gin.Context) {
	recipeID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid recipe ID",
			Message: "Recipe ID must be a number",
		})
		return
	}

	// Get user ID from context
	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "Unauthorized",
			Message: err.Error(),
		})
		return
	}

	// Check if recipe exists and belongs to user
	existingRecipe, err := h.recipeRepo.GetRecipeByID(c.Request.Context(), recipeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Message: err.Error(),
		})
		return
	}
	
	if existingRecipe == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Recipe not found",
			Message: "Recipe with the specified ID does not exist",
		})
		return
	}
	
	if existingRecipe.Recipe.UserID != userID {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error:   "Forbidden",
			Message: "You don't have permission to delete this recipe",
		})
		return
	}

	// Delete recipe
	err = h.recipeRepo.DeleteRecipe(c.Request.Context(), recipeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Message: err.Error(),
		})
		return
	}

	c.Status(http.StatusNoContent)
}

// GetMyRecipes handles GET /api/v1/recipes/my
func (h *RecipeHandler) GetMyRecipes(c *gin.Context) {
	// Get user ID from context
	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "Unauthorized",
			Message: err.Error(),
		})
		return
	}

	// Parse pagination parameters
	limit, offset, err := parsePaginationParams(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid pagination parameters",
			Message: err.Error(),
		})
		return
	}

	// Get user recipes
	recipes, total, err := h.recipeRepo.GetUserRecipes(c.Request.Context(), userID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Message: err.Error(),
		})
		return
	}

	// Build response
	response := model.RecipeSearchResponse{
		Data: recipes,
		Pagination: &model.Pagination{
			Page:       (offset / limit) + 1,
			Limit:      limit,
			Total:      total,
			TotalPages: (total + limit - 1) / limit,
		},
	}

	c.JSON(http.StatusOK, response)
}

// GetPublicRecipes handles GET /api/v1/recipes/public
func (h *RecipeHandler) GetPublicRecipes(c *gin.Context) {
	// Parse pagination parameters
	limit, offset, err := parsePaginationParams(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid pagination parameters",
			Message: err.Error(),
		})
		return
	}

	// Get public recipes
	recipes, total, err := h.recipeRepo.GetPublicRecipes(c.Request.Context(), limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Message: err.Error(),
		})
		return
	}

	// Build response
	response := model.RecipeSearchResponse{
		Data: recipes,
		Pagination: &model.Pagination{
			Page:       (offset / limit) + 1,
			Limit:      limit,
			Total:      total,
			TotalPages: (total + limit - 1) / limit,
		},
	}

	c.JSON(http.StatusOK, response)
}

// SearchRecipes handles GET /api/v1/recipes/search
func (h *RecipeHandler) SearchRecipes(c *gin.Context) {
	var req model.RecipeSearchRequest
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

	// Search recipes
	recipes, total, err := h.recipeRepo.SearchRecipes(c.Request.Context(), req.Query, req.PublicOnly, req.Limit, req.Offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Message: err.Error(),
		})
		return
	}

	// Build response
	response := model.RecipeSearchResponse{
		Data: recipes,
		Pagination: &model.Pagination{
			Page:       (req.Offset / req.Limit) + 1,
			Limit:      req.Limit,
			Total:      total,
			TotalPages: (total + req.Limit - 1) / req.Limit,
		},
	}

	c.JSON(http.StatusOK, response)
}

// AddRecipeToMyBook handles POST /api/v1/recipes/{id}/add-to-my-book
func (h *RecipeHandler) AddRecipeToMyBook(c *gin.Context) {
	recipeID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid recipe ID",
			Message: "Recipe ID must be a number",
		})
		return
	}

	// Get user ID from context
	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "Unauthorized",
			Message: err.Error(),
		})
		return
	}

	// Check if recipe exists and is public
	recipe, err := h.recipeRepo.GetRecipeByID(c.Request.Context(), recipeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Message: err.Error(),
		})
		return
	}
	
	if recipe == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Recipe not found",
			Message: "Recipe with the specified ID does not exist",
		})
		return
	}
	
	if !recipe.Recipe.IsPublic && recipe.Recipe.UserID != userID {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error:   "Forbidden",
			Message: "You don't have permission to add this recipe to your book",
		})
		return
	}

	// Add recipe to user's book
	err = h.recipeRepo.AddRecipeToUserBook(c.Request.Context(), userID, recipeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Recipe added to your book successfully",
	})
}

// GetMyRecipeBook handles GET /api/v1/recipes/my-book
func (h *RecipeHandler) GetMyRecipeBook(c *gin.Context) {
	// Get user ID from context
	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "Unauthorized",
			Message: err.Error(),
		})
		return
	}

	// Parse pagination parameters
	limit, offset, err := parsePaginationParams(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid pagination parameters",
			Message: err.Error(),
		})
		return
	}

	// Get user recipe book
	recipes, total, err := h.recipeRepo.GetUserRecipeBook(c.Request.Context(), userID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Message: err.Error(),
		})
		return
	}

	// Build response
	response := model.RecipeSearchResponse{
		Data: recipes,
		Pagination: &model.Pagination{
			Page:       (offset / limit) + 1,
			Limit:      limit,
			Total:      total,
			TotalPages: (total + limit - 1) / limit,
		},
	}

	c.JSON(http.StatusOK, response)
}

// parsePaginationParams parses limit and offset from query parameters
func parsePaginationParams(c *gin.Context) (int, int, error) {
	limit := 20
	offset := 0

	if limitStr := c.Query("limit"); limitStr != "" {
		l, err := strconv.Atoi(limitStr)
		if err != nil {
			return 0, 0, err
		}
		if l > 0 && l <= 100 {
			limit = l
		}
	}

	if offsetStr := c.Query("offset"); offsetStr != "" {
		o, err := strconv.Atoi(offsetStr)
		if err != nil {
			return 0, 0, err
		}
		if o >= 0 {
			offset = o
		}
	}

	return limit, offset, nil
}