package handler

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/n0dustoll3ns/nutrition-service/internal/model"
	"github.com/n0dustoll3ns/nutrition-service/internal/repository"
	"github.com/n0dustoll3ns/nutrition-service/internal/utils"
)

// MealPlanHandler handles meal plan-related HTTP requests
type MealPlanHandler struct {
	mealPlanRepo repository.MealPlanRepository
	diaryRepo    repository.DiaryRepository
	foodRepo     repository.FoodRepository
	db           *sql.DB
}

// NewMealPlanHandler creates a new MealPlanHandler
func NewMealPlanHandler(mealPlanRepo repository.MealPlanRepository, diaryRepo repository.DiaryRepository, foodRepo repository.FoodRepository, db *sql.DB) *MealPlanHandler {
	return &MealPlanHandler{
		mealPlanRepo: mealPlanRepo,
		diaryRepo:    diaryRepo,
		foodRepo:     foodRepo,
		db:           db,
	}
}

// CreateMealPlan handles POST /api/v1/meal-plans
// @Summary Create a new meal plan
// @Description Create a new meal plan with basic information
// @Tags meal-plans
// @Accept json
// @Produce json
// @Param request body model.MealPlanCreate true "Meal plan data"
// @Success 201 {object} model.MealPlan
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/meal-plans [post]
func (h *MealPlanHandler) CreateMealPlan(c *gin.Context) {
	var req model.MealPlanCreate
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

	// Create meal plan
	now := time.Now()
	mealPlan := &model.MealPlan{
		UserID:      userID,
		Name:        req.Name,
		Description: req.Description,
		DaysCount:   req.DaysCount,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	createdMealPlan, err := h.mealPlanRepo.CreateMealPlan(c.Request.Context(), mealPlan)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, createdMealPlan)
}

// GetMealPlans handles GET /api/v1/meal-plans
// @Summary Get user's meal plans
// @Description Get list of meal plans created by the user
// @Tags meal-plans
// @Accept json
// @Produce json
// @Param limit query int false "Number of items per page" default(20)
// @Param offset query int false "Offset for pagination" default(0)
// @Success 200 {object} model.MealPlanListResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/meal-plans [get]
func (h *MealPlanHandler) GetMealPlans(c *gin.Context) {
	var req struct {
		Limit  int `form:"limit,default=20"`
		Offset int `form:"offset,default=0"`
	}
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request parameters",
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

	// Get meal plans
	mealPlans, total, err := h.mealPlanRepo.GetUserMealPlans(c.Request.Context(), userID, req.Limit, req.Offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Message: err.Error(),
		})
		return
	}

	// Calculate pagination
	totalPages := 0
	if req.Limit > 0 {
		totalPages = (total + req.Limit - 1) / req.Limit
	}

	response := model.MealPlanListResponse{
		Data: mealPlans,
		Pagination: &model.Pagination{
			Page:       req.Offset/req.Limit + 1,
			Limit:      req.Limit,
			Total:      total,
			TotalPages: totalPages,
		},
	}

	c.JSON(http.StatusOK, response)
}

// GetMealPlan handles GET /api/v1/meal-plans/{id}
// @Summary Get meal plan by ID
// @Description Get meal plan with all its entries
// @Tags meal-plans
// @Accept json
// @Produce json
// @Param id path int true "Meal plan ID"
// @Success 200 {object} model.MealPlanWithEntries
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/meal-plans/{id} [get]
func (h *MealPlanHandler) GetMealPlan(c *gin.Context) {
	var uriParams struct {
		ID int `uri:"id" binding:"required"`
	}
	if err := c.ShouldBindUri(&uriParams); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid meal plan ID",
			Message: err.Error(),
		})
		return
	}

	// Get meal plan
	mealPlanWithEntries, err := h.mealPlanRepo.GetMealPlanByID(c.Request.Context(), uriParams.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Message: err.Error(),
		})
		return
	}

	if mealPlanWithEntries == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Meal plan not found",
			Message: "Meal plan with the specified ID does not exist",
		})
		return
	}

	// Check if meal plan belongs to user
	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "Unauthorized",
			Message: err.Error(),
		})
		return
	}

	if mealPlanWithEntries.MealPlan.UserID != userID {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error:   "Forbidden",
			Message: "You don't have permission to access this meal plan",
		})
		return
	}

	c.JSON(http.StatusOK, mealPlanWithEntries)
}

// UpdateMealPlan handles PUT /api/v1/meal-plans/{id}
// @Summary Update a meal plan
// @Description Update meal plan name and description
// @Tags meal-plans
// @Accept json
// @Produce json
// @Param id path int true "Meal plan ID"
// @Param request body model.MealPlanUpdate true "Update data"
// @Success 200 {object} model.MealPlan
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/meal-plans/{id} [put]
func (h *MealPlanHandler) UpdateMealPlan(c *gin.Context) {
	var uriParams struct {
		ID int `uri:"id" binding:"required"`
	}
	if err := c.ShouldBindUri(&uriParams); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid meal plan ID",
			Message: err.Error(),
		})
		return
	}

	var req model.MealPlanUpdate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request body",
			Message: err.Error(),
		})
		return
	}

	// Check if meal plan exists and belongs to user
	mealPlanWithEntries, err := h.mealPlanRepo.GetMealPlanByID(c.Request.Context(), uriParams.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Message: err.Error(),
		})
		return
	}

	if mealPlanWithEntries == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Meal plan not found",
			Message: "Meal plan with the specified ID does not exist",
		})
		return
	}

	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "Unauthorized",
			Message: err.Error(),
		})
		return
	}

	if mealPlanWithEntries.MealPlan.UserID != userID {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error:   "Forbidden",
			Message: "You don't have permission to update this meal plan",
		})
		return
	}

	// Update meal plan
	err = h.mealPlanRepo.UpdateMealPlan(c.Request.Context(), uriParams.ID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Message: err.Error(),
		})
		return
	}

	// Get updated meal plan
	updatedMealPlan, err := h.mealPlanRepo.GetMealPlanByID(c.Request.Context(), uriParams.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, updatedMealPlan.MealPlan)
}

// DeleteMealPlan handles DELETE /api/v1/meal-plans/{id}
// @Summary Delete a meal plan
// @Description Delete a meal plan and all its entries
// @Tags meal-plans
// @Accept json
// @Produce json
// @Param id path int true "Meal plan ID"
// @Success 204
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/meal-plans/{id} [delete]
func (h *MealPlanHandler) DeleteMealPlan(c *gin.Context) {
	var uriParams struct {
		ID int `uri:"id" binding:"required"`
	}
	if err := c.ShouldBindUri(&uriParams); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid meal plan ID",
			Message: err.Error(),
		})
		return
	}

	// Check if meal plan exists and belongs to user
	mealPlanWithEntries, err := h.mealPlanRepo.GetMealPlanByID(c.Request.Context(), uriParams.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Message: err.Error(),
		})
		return
	}

	if mealPlanWithEntries == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Meal plan not found",
			Message: "Meal plan with the specified ID does not exist",
		})
		return
	}

	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "Unauthorized",
			Message: err.Error(),
		})
		return
	}

	if mealPlanWithEntries.MealPlan.UserID != userID {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error:   "Forbidden",
			Message: "You don't have permission to delete this meal plan",
		})
		return
	}

	// Delete meal plan
	err = h.mealPlanRepo.DeleteMealPlan(c.Request.Context(), uriParams.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Message: err.Error(),
		})
		return
	}

	c.Status(http.StatusNoContent)
}

// CreateMealPlanEntry handles POST /api/v1/meal-plans/{id}/entries
// @Summary Add entry to meal plan
// @Description Add a food entry to a meal plan
// @Tags meal-plans
// @Accept json
// @Produce json
// @Param id path int true "Meal plan ID"
// @Param request body model.MealPlanEntryCreate true "Entry data"
// @Success 201 {object} model.MealPlanEntry
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/meal-plans/{id}/entries [post]
func (h *MealPlanHandler) CreateMealPlanEntry(c *gin.Context) {
	var uriParams struct {
		ID int `uri:"id" binding:"required"`
	}
	if err := c.ShouldBindUri(&uriParams); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid meal plan ID",
			Message: err.Error(),
		})
		return
	}

	var req model.MealPlanEntryCreate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request body",
			Message: err.Error(),
		})
		return
	}

	// Validate that exactly one of fdc_id, custom_food_name, or recipe_id is provided
	providedCount := 0
	if req.FDCID != nil {
		providedCount++
	}
	if req.CustomFoodName != nil {
		providedCount++
	}
	if req.RecipeID != nil {
		providedCount++
	}
	
	if providedCount != 1 {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Message: "Exactly one of fdc_id, custom_food_name, or recipe_id must be provided",
		})
		return
	}

	// Check if meal plan exists and belongs to user
	mealPlanWithEntries, err := h.mealPlanRepo.GetMealPlanByID(c.Request.Context(), uriParams.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Message: err.Error(),
		})
		return
	}

	if mealPlanWithEntries == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Meal plan not found",
			Message: "Meal plan with the specified ID does not exist",
		})
		return
	}

	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "Unauthorized",
			Message: err.Error(),
		})
		return
	}

	if mealPlanWithEntries.MealPlan.UserID != userID {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error:   "Forbidden",
			Message: "You don't have permission to add entries to this meal plan",
		})
		return
	}

	// Validate day number
	if req.DayNumber > mealPlanWithEntries.MealPlan.DaysCount {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid day number",
			Message: fmt.Sprintf("Day number must be between 1 and %d", mealPlanWithEntries.MealPlan.DaysCount),
		})
		return
	}

	// Create meal plan entry
	entry := &model.MealPlanEntry{
		ID:             uuid.New(),
		MealPlanID:     uriParams.ID,
		DayNumber:      req.DayNumber,
		MealType:       req.MealType,
		FDCID:          req.FDCID,
		CustomFoodName: req.CustomFoodName,
		RecipeID:       req.RecipeID,
		AmountGrams:    req.AmountGrams,
	}

	err = h.mealPlanRepo.CreateMealPlanEntry(c.Request.Context(), entry)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, entry)
}

// UpdateMealPlanEntry handles PUT /api/v1/meal-plans/{id}/entries/{entryId}
// @Summary Update a meal plan entry
// @Description Update a meal plan entry
// @Tags meal-plans
// @Accept json
// @Produce json
// @Param id path int true "Meal plan ID"
// @Param entryId path string true "Entry ID"
// @Param request body model.MealPlanEntryUpdate true "Update data"
// @Success 200 {object} model.MealPlanEntry
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/meal-plans/{id}/entries/{entryId} [put]
func (h *MealPlanHandler) UpdateMealPlanEntry(c *gin.Context) {
	var uriParams struct {
		ID      int    `uri:"id" binding:"required"`
		EntryID string `uri:"entryId" binding:"required"`
	}
	if err := c.ShouldBindUri(&uriParams); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid URI parameters",
			Message: err.Error(),
		})
		return
	}

	var req model.MealPlanEntryUpdate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request body",
			Message: err.Error(),
		})
		return
	}

	// Check if meal plan exists and belongs to user
	mealPlanWithEntries, err := h.mealPlanRepo.GetMealPlanByID(c.Request.Context(), uriParams.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Message: err.Error(),
		})
		return
	}

	if mealPlanWithEntries == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Meal plan not found",
			Message: "Meal plan with the specified ID does not exist",
		})
		return
	}

	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "Unauthorized",
			Message: err.Error(),
		})
		return
	}

	if mealPlanWithEntries.MealPlan.UserID != userID {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error:   "Forbidden",
			Message: "You don't have permission to update entries in this meal plan",
		})
		return
	}

	// Parse entry ID
	entryID, err := uuid.Parse(uriParams.EntryID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid entry ID",
			Message: "Entry ID must be a valid UUID",
		})
		return
	}

	// Validate day number if provided
	if req.DayNumber != nil && *req.DayNumber > mealPlanWithEntries.MealPlan.DaysCount {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid day number",
			Message: fmt.Sprintf("Day number must be between 1 and %d", mealPlanWithEntries.MealPlan.DaysCount),
		})
		return
	}

	// Update entry
	err = h.mealPlanRepo.UpdateMealPlanEntry(c.Request.Context(), entryID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Message: err.Error(),
		})
		return
	}

	// Get updated entry by finding it in the list
	// Note: In a real implementation, we might want a GetMealPlanEntryByID method
	updatedEntries, _, err := h.mealPlanRepo.GetMealPlanEntries(c.Request.Context(), uriParams.ID, 1000, 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Message: err.Error(),
		})
		return
	}

	var updatedEntry *model.MealPlanEntry
	for _, entry := range updatedEntries {
		if entry.ID == entryID {
			updatedEntry = entry
			break
		}
	}

	if updatedEntry == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Entry not found",
			Message: "Entry with the specified ID does not exist",
		})
		return
	}

	c.JSON(http.StatusOK, updatedEntry)
}

// DeleteMealPlanEntry handles DELETE /api/v1/meal-plans/{id}/entries/{entryId}
// @Summary Delete a meal plan entry
// @Description Delete a meal plan entry
// @Tags meal-plans
// @Accept json
// @Produce json
// @Param id path int true "Meal plan ID"
// @Param entryId path string true "Entry ID"
// @Success 204
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/meal-plans/{id}/entries/{entryId} [delete]
func (h *MealPlanHandler) DeleteMealPlanEntry(c *gin.Context) {
	var uriParams struct {
		ID      int    `uri:"id" binding:"required"`
		EntryID string `uri:"entryId" binding:"required"`
	}
	if err := c.ShouldBindUri(&uriParams); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid URI parameters",
			Message: err.Error(),
		})
		return
	}

	// Check if meal plan exists and belongs to user
	mealPlanWithEntries, err := h.mealPlanRepo.GetMealPlanByID(c.Request.Context(), uriParams.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Message: err.Error(),
		})
		return
	}

	if mealPlanWithEntries == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Meal plan not found",
			Message: "Meal plan with the specified ID does not exist",
		})
		return
	}

	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "Unauthorized",
			Message: err.Error(),
		})
		return
	}

	if mealPlanWithEntries.MealPlan.UserID != userID {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error:   "Forbidden",
			Message: "You don't have permission to delete entries from this meal plan",
		})
		return
	}

	// Parse entry ID
	entryID, err := uuid.Parse(uriParams.EntryID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid entry ID",
			Message: "Entry ID must be a valid UUID",
		})
		return
	}

	// Delete entry
	err = h.mealPlanRepo.DeleteMealPlanEntry(c.Request.Context(), entryID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Message: err.Error(),
		})
		return
	}

	c.Status(http.StatusNoContent)
}

// ApplyMealPlanToDiary handles POST /api/v1/meal-plans/{id}/apply-to-diary
// @Summary Apply meal plan to diary
// @Description Apply meal plan entries to diary for specific date
// @Tags meal-plans
// @Accept json
// @Produce json
// @Param id path int true "Meal plan ID"
// @Param request body model.MealPlanApplyRequest true "Apply parameters"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/meal-plans/{id}/apply-to-diary [post]
func (h *MealPlanHandler) ApplyMealPlanToDiary(c *gin.Context) {
	var uriParams struct {
		ID int `uri:"id" binding:"required"`
	}
	if err := c.ShouldBindUri(&uriParams); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid meal plan ID",
			Message: err.Error(),
		})
		return
	}

	var req model.MealPlanApplyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request body",
			Message: err.Error(),
		})
		return
	}

	// Parse start date
	startDate, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid date format",
			Message: "Start date must be in YYYY-MM-DD format",
		})
		return
	}

	// Check if meal plan exists and belongs to user
	mealPlanWithEntries, err := h.mealPlanRepo.GetMealPlanByID(c.Request.Context(), uriParams.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Message: err.Error(),
		})
		return
	}

	if mealPlanWithEntries == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Meal plan not found",
			Message: "Meal plan with the specified ID does not exist",
		})
		return
	}

	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "Unauthorized",
			Message: err.Error(),
		})
		return
	}

	if mealPlanWithEntries.MealPlan.UserID != userID {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error:   "Forbidden",
			Message: "You don't have permission to apply this meal plan",
		})
		return
	}

	// Get entries to apply
	var entriesToApply []*model.MealPlanEntry
	if req.DayNumber != nil && req.MealType != nil {
		// Apply specific meal type for specific day
		entries, err := h.mealPlanRepo.GetMealPlanEntriesByDayAndMeal(c.Request.Context(), uriParams.ID, *req.DayNumber, *req.MealType)
		if err != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error:   "Internal server error",
				Message: err.Error(),
			})
			return
		}
		entriesToApply = entries
	} else if req.DayNumber != nil {
		// Apply all meals for specific day
		entries, err := h.mealPlanRepo.GetMealPlanEntriesByDay(c.Request.Context(), uriParams.ID, *req.DayNumber)
		if err != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error:   "Internal server error",
				Message: err.Error(),
			})
			return
		}
		entriesToApply = entries
	} else {
		// Apply all entries (full plan)
		entriesToApply = mealPlanWithEntries.Entries
	}

	if len(entriesToApply) == 0 {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "No entries to apply",
			Message: "The selected meal plan or day/meal has no entries",
		})
		return
	}

	// Calculate target date
	targetDate := startDate
	if req.DayNumber != nil {
		// Apply specific day (day 1 = startDate, day 2 = startDate + 1 day, etc.)
		targetDate = startDate.AddDate(0, 0, *req.DayNumber-1)
	}

	// Convert meal plan entries to diary entries
	var diaryEntries []*model.FoodEntry
	nutrientCalculator := utils.NewNutrientCalculator(h.db)

	for _, mealPlanEntry := range entriesToApply {
		// Calculate nutrients
		calculatedNutrients, err := nutrientCalculator.CalculateNutrientsWithFallback(
			c.Request.Context(),
			mealPlanEntry.FDCID,
			nil, // custom calories
			nil, // custom protein
			nil, // custom fat
			nil, // custom carbs
			mealPlanEntry.AmountGrams,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error:   "Failed to calculate nutrients",
				Message: err.Error(),
			})
			return
		}

		// Create diary entry
		diaryEntry := &model.FoodEntry{
			ID:                 uuid.New(),
			UserID:             userID,
			Date:               targetDate,
			MealType:           mealPlanEntry.MealType,
			FDCID:              mealPlanEntry.FDCID,
			CustomFoodName:     mealPlanEntry.CustomFoodName,
			RecipeID:           mealPlanEntry.RecipeID,
			AmountGrams:        mealPlanEntry.AmountGrams,
			CalculatedCalories: calculatedNutrients.Calories,
			CalculatedProtein:  calculatedNutrients.Protein,
			CalculatedFat:      calculatedNutrients.Fat,
			CalculatedCarbs:    calculatedNutrients.Carbs,
			CreatedAt:          time.Now(),
		}

		diaryEntries = append(diaryEntries, diaryEntry)
	}

	// Create diary entries in batch
	err = h.diaryRepo.CreateFoodEntries(c.Request.Context(), diaryEntries)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to create diary entries",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "Meal plan applied to diary successfully",
		"entries_count": len(diaryEntries),
		"date":         targetDate.Format("2006-01-02"),
		"day_number":   req.DayNumber,
		"meal_type":    req.MealType,
	})
}
