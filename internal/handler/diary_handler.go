package handler

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/yourusername/auth-service/internal/model"
	"github.com/yourusername/auth-service/internal/repository"
	"github.com/yourusername/auth-service/internal/utils"
)

// DiaryHandler handles diary-related HTTP requests
type DiaryHandler struct {
	diaryRepo repository.DiaryRepository
	foodRepo  repository.FoodRepository
	db        *sql.DB
}

// NewDiaryHandler creates a new DiaryHandler
func NewDiaryHandler(diaryRepo repository.DiaryRepository, foodRepo repository.FoodRepository, db *sql.DB) *DiaryHandler {
	return &DiaryHandler{
		diaryRepo: diaryRepo,
		foodRepo:  foodRepo,
		db:        db,
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

	// Load food names for entries with FDCID
	// Collect unique FDCIDs
	fdcIDs := make(map[int]bool)
	for _, entry := range entries {
		if entry.FDCID != nil {
			fdcIDs[*entry.FDCID] = true
		}
	}

	// Load food names from repository in parallel for better performance
	type foodResult struct {
		fdcID int
		name  string
		err   error
	}
	
	resultChan := make(chan foodResult, len(fdcIDs))
	
	// Launch goroutines to load food names
	for fdcID := range fdcIDs {
		go func(id int) {
			food, err := h.foodRepo.GetFoodByID(c.Request.Context(), id)
			if err != nil {
				resultChan <- foodResult{fdcID: id, err: err}
				return
			}
			if food != nil && food.Food != nil {
				resultChan <- foodResult{fdcID: id, name: food.Food.Description}
			} else {
				resultChan <- foodResult{fdcID: id}
			}
		}(fdcID)
	}
	
	// Collect results
	foodNames := make(map[int]string)
	for i := 0; i < len(fdcIDs); i++ {
		result := <-resultChan
		if result.err != nil {
			// Log error but continue - we'll just skip this food name
			fmt.Printf("Failed to load food name for FDCID %d: %v\n", result.fdcID, result.err)
		} else if result.name != "" {
			foodNames[result.fdcID] = result.name
		}
	}
	close(resultChan)

	// Set FoodName for each entry
	for _, entry := range entries {
		if entry.FDCID != nil {
			if name, ok := foodNames[*entry.FDCID]; ok {
				entry.FoodName = &name
			}
		} else if entry.CustomFoodName != nil {
			// For custom foods, use the custom food name
			entry.FoodName = entry.CustomFoodName
		}
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
// @Summary Create food entries
// @Description Add one or multiple food entries to the diary
// @Tags diary
// @Accept json
// @Produce json
// @Param request body any true "Food entry data (single object or array)"
// @Success 201 {object} any "Single FoodEntry or array of FoodEntry"
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/diary/entries [post]
func (h *DiaryHandler) CreateFoodEntry(c *gin.Context) {
	// Try to parse as array first
	var reqArray []model.FoodEntryCreate
	if err := c.ShouldBindJSON(&reqArray); err == nil {
		// Successfully parsed as array
		h.createFoodEntriesBatch(c, reqArray)
		return
	}

	// Try to parse as single object
	var reqSingle model.FoodEntryCreate
	if err := c.ShouldBindJSON(&reqSingle); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request body",
			Message: "Request must be a single food entry object or an array of food entries",
		})
		return
	}

	// Single entry - wrap in array and process
	h.createFoodEntriesBatch(c, []model.FoodEntryCreate{reqSingle})
}

// createFoodEntriesBatch processes multiple food entries in batch
func (h *DiaryHandler) createFoodEntriesBatch(c *gin.Context, reqs []model.FoodEntryCreate) {
	if len(reqs) == 0 {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Empty request",
			Message: "At least one food entry must be provided",
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

	// Create nutrient calculator
	nutrientCalculator := utils.NewNutrientCalculator(h.db)

	var diaryEntries []*model.FoodEntry
	var fdcIDsToVerify []int

	// First pass: validate and prepare entries
	for i, req := range reqs {
		// Validate that exactly one of fdc_id, custom_food_name, or recipe_id is provided
		providedCount := 0
		if req.FDCID != nil {
			providedCount++
			fdcIDsToVerify = append(fdcIDsToVerify, *req.FDCID)
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
				Message: fmt.Sprintf("Entry at position %d: exactly one of fdc_id, custom_food_name, or recipe_id must be provided", i),
			})
			return
		}

		// Parse date
		date, err := time.Parse("2006-01-02", req.Date)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "Invalid date format",
				Message: fmt.Sprintf("Entry at position %d: date must be in YYYY-MM-DD format", i),
			})
			return
		}

		// Calculate nutrients using the nutrient calculator
		calculatedNutrients, err := nutrientCalculator.CalculateNutrientsWithFallback(
			c.Request.Context(),
			req.FDCID,
			req.CustomCalories,
			req.CustomProtein,
			req.CustomFat,
			req.CustomCarbs,
			req.AmountGrams,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error:   "Failed to calculate nutrients",
				Message: fmt.Sprintf("Entry at position %d: %s", i, err.Error()),
			})
			return
		}

		// Create food entry
		entry := &model.FoodEntry{
			ID:                 uuid.New(),
			UserID:             userID,
			Date:               date,
			MealType:           req.MealType,
			FDCID:              req.FDCID,
			CustomFoodName:     req.CustomFoodName,
			RecipeID:           req.RecipeID,
			AmountGrams:        req.AmountGrams,
			CalculatedCalories: calculatedNutrients.Calories,
			CalculatedProtein:  calculatedNutrients.Protein,
			CalculatedFat:      calculatedNutrients.Fat,
			CalculatedCarbs:    calculatedNutrients.Carbs,
			CreatedAt:          time.Now(),
		}

		diaryEntries = append(diaryEntries, entry)
	}

	// Verify FDC IDs exist
	if len(fdcIDsToVerify) > 0 {
		uniqueFDCIDs := make(map[int]bool)
		for _, fdcID := range fdcIDsToVerify {
			uniqueFDCIDs[fdcID] = true
		}

		for fdcID := range uniqueFDCIDs {
			foodWithNutrients, err := h.foodRepo.GetFoodByID(c.Request.Context(), fdcID)
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
					Message: fmt.Sprintf("Food with FDC ID %d does not exist", fdcID),
				})
				return
			}
		}
	}

	// Set FoodName for entries with FDCID
	// Collect unique FDCIDs for batch loading
	fdcIDMap := make(map[int]bool)
	for _, entry := range diaryEntries {
		if entry.FDCID != nil {
			fdcIDMap[*entry.FDCID] = true
		}
	}

	// Load food names in batch
	foodNames := make(map[int]string)
	for fdcID := range fdcIDMap {
		food, err := h.foodRepo.GetFoodByID(c.Request.Context(), fdcID)
		if err == nil && food != nil && food.Food != nil {
			foodNames[fdcID] = food.Food.Description
		}
	}

	// Set FoodName for each entry
	for _, entry := range diaryEntries {
		if entry.FDCID != nil {
			if name, ok := foodNames[*entry.FDCID]; ok {
				entry.FoodName = &name
			}
		} else if entry.CustomFoodName != nil {
			// For custom foods, use the custom food name
			entry.FoodName = entry.CustomFoodName
		}
	}

	// Save to database
	if len(diaryEntries) == 1 {
		// Use single insert for better error messages
		err = h.diaryRepo.CreateFoodEntry(c.Request.Context(), diaryEntries[0])
	} else {
		// Use batch insert
		err = h.diaryRepo.CreateFoodEntries(c.Request.Context(), diaryEntries)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Message: err.Error(),
		})
		return
	}

	// Return appropriate response
	if len(diaryEntries) == 1 {
		c.JSON(http.StatusCreated, diaryEntries[0])
	} else {
		c.JSON(http.StatusCreated, diaryEntries)
	}
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

	// Set FoodName for the updated entry
	if updatedEntry != nil {
		if updatedEntry.FDCID != nil {
			// Load food name from repository
			food, err := h.foodRepo.GetFoodByID(c.Request.Context(), *updatedEntry.FDCID)
			if err == nil && food != nil && food.Food != nil {
				foodName := food.Food.Description
				updatedEntry.FoodName = &foodName
			}
		} else if updatedEntry.CustomFoodName != nil {
			// For custom foods, use the custom food name
			updatedEntry.FoodName = updatedEntry.CustomFoodName
		}
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
