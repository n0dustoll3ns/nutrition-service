package handler

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/yourusername/auth-service/internal/model"
	"github.com/yourusername/auth-service/internal/repository"
)

// DiaryHandler handles diary-related HTTP requests
type DiaryHandler struct {
	diaryRepo repository.DiaryRepository
	foodRepo  repository.FoodRepository
}

// NewDiaryHandler creates a new DiaryHandler
func NewDiaryHandler(diaryRepo repository.DiaryRepository, foodRepo repository.FoodRepository) *DiaryHandler {
	return &DiaryHandler{
		diaryRepo: diaryRepo,
		foodRepo:  foodRepo,
	}
}

// GetDiaryEntries handles GET /api/v1/diary/entries
// @Summary Get diary entries for a period
// @Description Get food entries for a user within a date period
// @Tags diary
// @Accept json
// @Produce json
// @Param date query string true "Base date (YYYY-MM-DD)"
// @Param daysCount query int false "Number of days to include (default: 1)" default(1)
// @Success 200 {object} model.DiaryPeriodResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/diary/entries [get]
func (h *DiaryHandler) GetDiaryEntries(c *gin.Context) {
	var req model.DiaryPeriodRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request parameters",
			Message: err.Error(),
		})
		return
	}

	// Parse dates
	baseDate, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid date format",
			Message: "Date must be in YYYY-MM-DD format",
		})
		return
	}

	// Calculate period
	endDate := baseDate
	startDate := baseDate.AddDate(0, 0, -(req.DaysCount - 1))

	// Get user ID from context (set by auth middleware)
	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "Unauthorized",
			Message: err.Error(),
		})
		return
	}

	// Get food entries for the period
	entries, err := h.diaryRepo.GetFoodEntriesByPeriod(c.Request.Context(), userID, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Message: err.Error(),
		})
		return
	}

	// Organize entries by date and meal type
	daysMap := make(map[string]*model.DiaryDay)
	mealTypes := model.MealTypes()

	// Initialize all days in the period with empty meal structures
	for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
		dateStr := d.Format("2006-01-02")
		meals := make(map[string][]*model.FoodEntry)
		for _, mealType := range mealTypes {
			meals[mealType] = []*model.FoodEntry{}
		}
		
		daysMap[dateStr] = &model.DiaryDay{
			Date:  d,
			Meals: meals,
		}
	}

	// Populate with actual entries
	for _, entry := range entries {
		dateStr := entry.Date.Format("2006-01-02")
		if day, exists := daysMap[dateStr]; exists {
			day.Meals[entry.MealType] = append(day.Meals[entry.MealType], entry)
		}
	}

	// Convert map to sorted slice and calculate summaries
	var days []*model.DiaryDay
	for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
		dateStr := d.Format("2006-01-02")
		day := daysMap[dateStr]
		
		// Calculate summary for the day
		summary, err := h.diaryRepo.GetDaySummary(c.Request.Context(), userID, d)
		if err != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error:   "Internal server error",
				Message: err.Error(),
			})
			return
		}
		day.Summary = summary
		
		days = append(days, day)
	}

	// Build response
	response := model.DiaryPeriodResponse{
		Period: struct {
			StartDate string `json:"start_date"`
			EndDate   string `json:"end_date"`
		}{
			StartDate: startDate.Format("2006-01-02"),
			EndDate:   endDate.Format("2006-01-02"),
		},
		Days: days,
	}

	c.JSON(http.StatusOK, response)
}

// CreateFoodEntry handles POST /api/v1/diary/entries
// @Summary Create a new food entry
// @Description Add a food entry to the diary
// @Tags diary
// @Accept json
// @Produce json
// @Param request body model.FoodEntryCreate true "Food entry data"
// @Success 201 {object} model.FoodEntry
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/diary/entries [post]
func (h *DiaryHandler) CreateFoodEntry(c *gin.Context) {
	var req model.FoodEntryCreate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request body",
			Message: err.Error(),
		})
		return
	}

	// Validate that exactly one of fdc_id or custom_food_name is provided
	if (req.FDCID == nil && req.CustomFoodName == nil) || (req.FDCID != nil && req.CustomFoodName != nil) {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Message: "Exactly one of fdc_id or custom_food_name must be provided",
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

	// Parse date
	date, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid date format",
			Message: "Date must be in YYYY-MM-DD format",
		})
		return
	}

	// Calculate nutrients
	var calculatedCalories, calculatedProtein, calculatedFat, calculatedCarbs *float64
	
	if req.FDCID != nil {
		// Get food from USDA database and calculate nutrients
		foodWithNutrients, err := h.foodRepo.GetFoodByID(c.Request.Context(), *req.FDCID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error:   "Internal server error",
				Message: err.Error(),
			})
			return
		}
		
		if foodWithNutrients == nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "Food not found",
				Message: "Food with the specified FDC ID does not exist",
			})
			return
		}
		
		// Calculate nutrients based on amount_grams
		// This is a simplified calculation - in reality you'd need to find
		// the specific nutrients (calories, protein, fat, carbs) from the nutrients list
		// For now, we'll use custom values if provided, otherwise set to nil
		if req.CustomCalories != nil {
			calculatedCalories = req.CustomCalories
		}
		if req.CustomProtein != nil {
			calculatedProtein = req.CustomProtein
		}
		if req.CustomFat != nil {
			calculatedFat = req.CustomFat
		}
		if req.CustomCarbs != nil {
			calculatedCarbs = req.CustomCarbs
		}
	} else {
		// Custom food - use provided custom values
		calculatedCalories = req.CustomCalories
		calculatedProtein = req.CustomProtein
		calculatedFat = req.CustomFat
		calculatedCarbs = req.CustomCarbs
	}

	// Create food entry
	entry := &model.FoodEntry{
		ID:                 uuid.New(),
		UserID:             userID,
		Date:               date,
		MealType:           req.MealType,
		FDCID:              req.FDCID,
		CustomFoodName:     req.CustomFoodName,
		AmountGrams:        req.AmountGrams,
		CalculatedCalories: calculatedCalories,
		CalculatedProtein:  calculatedProtein,
		CalculatedFat:      calculatedFat,
		CalculatedCarbs:    calculatedCarbs,
		CreatedAt:          time.Now(),
	}

	// Save to database
	err = h.diaryRepo.CreateFoodEntry(c.Request.Context(), entry)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, entry)
}

// UpdateFoodEntry handles PUT /api/v1/diary/entries/{id}
// @Summary Update a food entry
// @Description Update an existing food entry
// @Tags diary
// @Accept json
// @Produce json
// @Param id path string true "Food entry ID"
// @Param request body model.FoodEntryUpdate true "Update data"
// @Success 200 {object} model.FoodEntry
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/diary/entries/{id} [put]
func (h *DiaryHandler) UpdateFoodEntry(c *gin.Context) {
	var uriParams struct {
		ID string `uri:"id" binding:"required"`
	}
	if err := c.ShouldBindUri(&uriParams); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid food entry ID",
			Message: err.Error(),
		})
		return
	}

	var req model.FoodEntryUpdate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request body",
			Message: err.Error(),
		})
		return
	}

	// Parse UUID
	entryID, err := uuid.Parse(uriParams.ID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid food entry ID",
			Message: "ID must be a valid UUID",
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

	// Check if entry exists and belongs to user
	existingEntry, err := h.diaryRepo.GetFoodEntryByID(c.Request.Context(), entryID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Message: err.Error(),
		})
		return
	}
	
	if existingEntry == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Food entry not found",
			Message: "Food entry with the specified ID does not exist",
		})
		return
	}
	
	if existingEntry.UserID != userID {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error:   "Forbidden",
			Message: "You don't have permission to update this food entry",
		})
		return
	}

	// Update entry
	err = h.diaryRepo.UpdateFoodEntry(c.Request.Context(), entryID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Message: err.Error(),
		})
		return
	}

	// Get updated entry
	updatedEntry, err := h.diaryRepo.GetFoodEntryByID(c.Request.Context(), entryID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, updatedEntry)
}

// DeleteFoodEntry handles DELETE /api/v1/diary/entries/{id}
// @Summary Delete a food entry
// @Description Delete an existing food entry
// @Tags diary
// @Accept json
// @Produce json
// @Param id path string true "Food entry ID"
// @Success 204
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/diary/entries/{id} [delete]
func (h *DiaryHandler) DeleteFoodEntry(c *gin.Context) {
	var uriParams struct {
		ID string `uri:"id" binding:"required"`
	}
	if err := c.ShouldBindUri(&uriParams); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid food entry ID",
			Message: err.Error(),
		})
		return
	}

	// Parse UUID
	entryID, err := uuid.Parse(uriParams.ID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid food entry ID",
			Message: "ID must be a valid UUID",
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

	// Check if entry exists and belongs to user
	existingEntry, err := h.diaryRepo.GetFoodEntryByID(c.Request.Context(), entryID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Message: err.Error(),
		})
		return
	}
	
	if existingEntry == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Food entry not found",
			Message: "Food entry with the specified ID does not exist",
		})
		return
	}
	
	if existingEntry.UserID != userID {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error:   "Forbidden",
			Message: "You don't have permission to delete this food entry",
		})
		return
	}

	// Delete entry
	err = h.diaryRepo.DeleteFoodEntry(c.Request.Context(), entryID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Message: err.Error(),
		})
		return
	}

	c.Status(http.StatusNoContent)
}

// GetDiarySummary handles GET /api/v1/diary/summary
// @Summary Get diary summary for a period
// @Description Get nutritional summary for a user within a date period
// @Tags diary
// @Accept json
// @Produce json
// @Param date query string true "Base date (YYYY-MM-DD)"
// @Param daysCount query int false "Number of days to include (default: 1)" default(1)
// @Success 200 {object} model.DiarySummaryResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/diary/summary [get]
func (h *DiaryHandler) GetDiarySummary(c *gin.Context) {
	var req model.DiarySummaryRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request parameters",
			Message: err.Error(),
		})
		return
	}

	// Parse dates
	baseDate, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid date format",
			Message: "Date must be in YYYY-MM-DD format",
		})
		return
	}

	// Calculate period
	endDate := baseDate
	startDate := baseDate.AddDate(0, 0, -(req.DaysCount - 1))

	// Get user ID from context
	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "Unauthorized",
			Message: err.Error(),
		})
		return
	}

	// Get period summary
	summary, err := h.diaryRepo.GetPeriodSummary(c.Request.Context(), userID, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Message: err.Error(),
		})
		return
	}

	// Build response
	response := model.DiarySummaryResponse{
		Period: struct {
			StartDate string `json:"start_date"`
			EndDate   string `json:"end_date"`
		}{
			StartDate: startDate.Format("2006-01-02"),
			EndDate:   endDate.Format("2006-01-02"),
		},
		Summary: summary,
	}

	c.JSON(http.StatusOK, response)
}

// CopyDiaryEntries handles POST /api/v1/diary/copy
// @Summary Copy diary entries
// @Description Copy food entries from one date to another
// @Tags diary
// @Accept json
// @Produce json
// @Param request body model.DiaryCopyRequest true "Copy parameters"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/diary/copy [post]
func (h *DiaryHandler) CopyDiaryEntries(c *gin.Context) {
	var req model.DiaryCopyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request body",
			Message: err.Error(),
		})
		return
	}

	// Parse dates
	sourceDate, err := time.Parse("2006-01-02", req.SourceDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid source date format",
			Message: "Date must be in YYYY-MM-DD format",
		})
		return
	}

	targetDate, err := time.Parse("2006-01-02", req.TargetDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid target date format",
			Message: "Date must be in YYYY-MM-DD format",
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

	// Copy entries
	err = h.diaryRepo.CopyFoodEntries(c.Request.Context(), userID, sourceDate, targetDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Diary entries copied successfully",
		"source_date": req.SourceDate,
		"target_date": req.TargetDate,
	})
}

// getUserIDFromContext extracts user ID from Gin context (set by auth middleware)
func getUserIDFromContext(c *gin.Context) (uuid.UUID, error) {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		return uuid.Nil, fmt.Errorf("user ID not found in context")
	}

	userIDStr, ok := userIDVal.(string)
	if !ok {
		return uuid.Nil, fmt.Errorf("invalid user ID type in context")
	}

	return uuid.Parse(userIDStr)
}
