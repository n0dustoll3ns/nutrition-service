package model

import (
	"time"

	"github.com/google/uuid"
)

// FoodEntry represents a food entry in the diary
type FoodEntry struct {
	ID                 uuid.UUID  `json:"id" db:"id"`
	UserID             uuid.UUID  `json:"user_id" db:"user_id"`
	Date               time.Time  `json:"date" db:"date"`
	MealType           string     `json:"meal_type" db:"meal_type"`
	FDCID              *int       `json:"fdc_id,omitempty" db:"fdc_id"`
	CustomFoodName     *string    `json:"custom_food_name,omitempty" db:"custom_food_name"`
	AmountGrams        float64    `json:"amount_grams" db:"amount_grams"`
	CalculatedCalories *float64   `json:"calculated_calories,omitempty" db:"calculated_calories"`
	CalculatedProtein  *float64   `json:"calculated_protein,omitempty" db:"calculated_protein"`
	CalculatedFat      *float64   `json:"calculated_fat,omitempty" db:"calculated_fat"`
	CalculatedCarbs    *float64   `json:"calculated_carbs,omitempty" db:"calculated_carbs"`
	CreatedAt          time.Time  `json:"created_at" db:"created_at"`
}

// FoodEntryCreate represents data needed to create a new food entry
type FoodEntryCreate struct {
	Date            string   `json:"date" binding:"required,datetime=2006-01-02"`
	MealType        string   `json:"meal_type" binding:"required,oneof=breakfast brunch lunch afternoon_snack dinner snack"`
	FDCID           *int     `json:"fdc_id,omitempty"`
	CustomFoodName  *string  `json:"custom_food_name,omitempty"`
	AmountGrams     float64  `json:"amount_grams" binding:"required,gt=0"`
	CustomCalories  *float64 `json:"custom_calories,omitempty"`
	CustomProtein   *float64 `json:"custom_protein,omitempty"`
	CustomFat       *float64 `json:"custom_fat,omitempty"`
	CustomCarbs     *float64 `json:"custom_carbs,omitempty"`
}

// FoodEntryUpdate represents data needed to update a food entry
type FoodEntryUpdate struct {
	AmountGrams    *float64 `json:"amount_grams,omitempty" binding:"omitempty,gt=0"`
	CustomCalories *float64 `json:"custom_calories,omitempty"`
	CustomProtein  *float64 `json:"custom_protein,omitempty"`
	CustomFat      *float64 `json:"custom_fat,omitempty"`
	CustomCarbs    *float64 `json:"custom_carbs,omitempty"`
}

// DiaryDay represents a day in the diary with all meals
type DiaryDay struct {
	Date   time.Time               `json:"date"`
	Meals  map[string][]*FoodEntry `json:"meals"` // key: meal_type
	Summary *DaySummary            `json:"summary,omitempty"`
}

// DaySummary represents nutritional summary for a day
type DaySummary struct {
	TotalCalories float64 `json:"total_calories"`
	TotalProtein  float64 `json:"total_protein"`
	TotalFat      float64 `json:"total_fat"`
	TotalCarbs    float64 `json:"total_carbs"`
	MealCount     int     `json:"meal_count"`
	FoodCount     int     `json:"food_count"`
}

// DiaryPeriodRequest represents request parameters for getting diary entries
type DiaryPeriodRequest struct {
	Date      string `form:"date" binding:"required,datetime=2006-01-02"`
	DaysCount int    `form:"daysCount,default=1"`
}

// DiaryPeriodResponse represents response with diary entries for a period
type DiaryPeriodResponse struct {
	Period struct {
		StartDate string `json:"start_date"`
		EndDate   string `json:"end_date"`
	} `json:"period"`
	Days []*DiaryDay `json:"days"`
}

// DiarySummaryRequest represents request parameters for getting diary summary
type DiarySummaryRequest struct {
	Date      string `form:"date" binding:"required,datetime=2006-01-02"`
	DaysCount int    `form:"daysCount,default=1"`
}

// DiarySummaryResponse represents nutritional summary for a period
type DiarySummaryResponse struct {
	Period struct {
		StartDate string `json:"start_date"`
		EndDate   string `json:"end_date"`
	} `json:"period"`
	Summary *DaySummary `json:"summary"`
}

// DiaryCopyRequest represents request to copy diary entries
type DiaryCopyRequest struct {
	SourceDate string `json:"source_date" binding:"required,datetime=2006-01-02"`
	TargetDate string `json:"target_date" binding:"required,datetime=2006-01-02"`
	CopyAll    bool   `json:"copy_all,default=true"`
}

// MealTypes returns the list of valid meal types
func MealTypes() []string {
	return []string{"breakfast", "brunch", "lunch", "afternoon_snack", "dinner", "snack"}
}

// IsValidMealType checks if a meal type is valid
func IsValidMealType(mealType string) bool {
	for _, mt := range MealTypes() {
		if mt == mealType {
			return true
		}
	}
	return false
}