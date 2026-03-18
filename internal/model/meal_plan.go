package model

import (
	"time"

	"github.com/google/uuid"
)

// MealPlan represents a meal plan
type MealPlan struct {
	ID          int        `json:"id" db:"id"`
	UserID      uuid.UUID  `json:"user_id" db:"user_id"`
	Name        string     `json:"name" db:"name"`
	Description *string    `json:"description,omitempty" db:"description"`
	DaysCount   int        `json:"days_count" db:"days_count"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
}

// MealPlanEntry represents an entry in a meal plan
type MealPlanEntry struct {
	ID             uuid.UUID `json:"id" db:"id"`
	MealPlanID     int       `json:"meal_plan_id" db:"meal_plan_id"`
	DayNumber      int       `json:"day_number" db:"day_number"`
	MealType       string    `json:"meal_type" db:"meal_type"`
	FDCID          *int      `json:"fdc_id,omitempty" db:"fdc_id"`
	CustomFoodName *string   `json:"custom_food_name,omitempty" db:"custom_food_name"`
	RecipeID       *int      `json:"recipe_id,omitempty" db:"recipe_id"`
	AmountGrams    float64   `json:"amount_grams" db:"amount_grams"`
}

// MealPlanWithEntries represents a meal plan with its entries
type MealPlanWithEntries struct {
	MealPlan *MealPlan        `json:"meal_plan"`
	Entries  []*MealPlanEntry `json:"entries"`
}

// MealPlanCreate represents data needed to create a new meal plan
type MealPlanCreate struct {
	Name        string  `json:"name" binding:"required"`
	Description *string `json:"description,omitempty"`
	DaysCount   int     `json:"days_count" binding:"required,gt=0"`
}

// MealPlanUpdate represents data needed to update a meal plan
type MealPlanUpdate struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
}

// MealPlanEntryCreate represents data needed to create a meal plan entry
type MealPlanEntryCreate struct {
	DayNumber      int     `json:"day_number" binding:"required,gt=0"`
	MealType       string  `json:"meal_type" binding:"required,oneof=breakfast brunch lunch afternoon_snack dinner snack"`
	FDCID          *int    `json:"fdc_id,omitempty"`
	CustomFoodName *string `json:"custom_food_name,omitempty"`
	RecipeID       *int    `json:"recipe_id,omitempty"`
	AmountGrams    float64 `json:"amount_grams" binding:"required,gt=0"`
}

// MealPlanEntryUpdate represents data needed to update a meal plan entry
type MealPlanEntryUpdate struct {
	DayNumber   *int     `json:"day_number,omitempty" binding:"omitempty,gt=0"`
	MealType    *string  `json:"meal_type,omitempty" binding:"omitempty,oneof=breakfast brunch lunch afternoon_snack dinner snack"`
	AmountGrams *float64 `json:"amount_grams,omitempty" binding:"omitempty,gt=0"`
}

// MealPlanApplyRequest represents request to apply meal plan to diary
type MealPlanApplyRequest struct {
	StartDate   string `json:"start_date" binding:"required,datetime=2006-01-02"`
	DayNumber   *int   `json:"day_number,omitempty" binding:"omitempty,gt=0"`
	MealType    *string `json:"meal_type,omitempty" binding:"omitempty,oneof=breakfast brunch lunch afternoon_snack dinner snack"`
}

// MealPlanListResponse represents response for listing meal plans
type MealPlanListResponse struct {
	Data       []*MealPlan `json:"data"`
	Pagination *Pagination `json:"pagination"`
}

// MealPlanEntriesResponse represents response for meal plan entries
type MealPlanEntriesResponse struct {
	MealPlanID int               `json:"meal_plan_id"`
	Entries    []*MealPlanEntry  `json:"entries"`
	Pagination *Pagination       `json:"pagination"`
}