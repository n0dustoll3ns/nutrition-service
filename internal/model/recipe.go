package model

import (
	"time"

	"github.com/google/uuid"
)

// Recipe represents a user-created recipe
type Recipe struct {
	ID          int        `json:"id" db:"id"`
	UserID      uuid.UUID  `json:"user_id" db:"user_id"`
	Name        string     `json:"name" db:"name"`
	Description *string    `json:"description,omitempty" db:"description"`
	Category    *string    `json:"category,omitempty" db:"category"`
	IsPublic    bool       `json:"is_public" db:"is_public"`
	TotalWeight float64    `json:"total_weight" db:"total_weight"` // Total weight of recipe in grams
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
}

// RecipeIngredient represents an ingredient in a recipe
type RecipeIngredient struct {
	ID              uuid.UUID `json:"id" db:"id"`
	RecipeID        int       `json:"recipe_id" db:"recipe_id"`
	FDCID           *int      `json:"fdc_id,omitempty" db:"fdc_id"`
	CustomFoodName  *string   `json:"custom_food_name,omitempty" db:"custom_food_name"`
	AmountGrams     float64   `json:"amount_grams" db:"amount_grams"`
	// Cached nutrients per 100g of this ingredient in the recipe
	CaloriesPer100g *float64 `json:"calories_per_100g,omitempty" db:"calories_per_100g"`
	ProteinPer100g  *float64 `json:"protein_per_100g,omitempty" db:"protein_per_100g"`
	FatPer100g      *float64 `json:"fat_per_100g,omitempty" db:"fat_per_100g"`
	CarbsPer100g    *float64 `json:"carbs_per_100g,omitempty" db:"carbs_per_100g"`
}

// RecipeWithIngredients represents a recipe with its ingredients
type RecipeWithIngredients struct {
	Recipe      *Recipe             `json:"recipe"`
	Ingredients []*RecipeIngredient `json:"ingredients"`
}

// UserRecipe represents a recipe added by a user to their recipe book
type UserRecipe struct {
	UserID   uuid.UUID `json:"user_id" db:"user_id"`
	RecipeID int       `json:"recipe_id" db:"recipe_id"`
	AddedAt  time.Time `json:"added_at" db:"added_at"`
}

// RecipeCreate represents data needed to create a new recipe
type RecipeCreate struct {
	Name        string  `json:"name" binding:"required"`
	Description *string `json:"description,omitempty"`
	Category    *string `json:"category,omitempty"`
	IsPublic    bool    `json:"is_public"`
	Ingredients []RecipeIngredientCreate `json:"ingredients" binding:"required,min=1"`
}

// RecipeIngredientCreate represents data needed to create a recipe ingredient
type RecipeIngredientCreate struct {
	FDCID          *int    `json:"fdc_id,omitempty"`
	CustomFoodName *string `json:"custom_food_name,omitempty"`
	AmountGrams    float64 `json:"amount_grams" binding:"required,gt=0"`
}

// RecipeUpdate represents data needed to update a recipe
type RecipeUpdate struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	Category    *string `json:"category,omitempty"`
	IsPublic    *bool   `json:"is_public,omitempty"`
}

// RecipeSearchRequest represents request parameters for searching recipes
type RecipeSearchRequest struct {
	Query  string `form:"q" binding:"required"`
	Limit  int    `form:"limit,default=20"`
	Offset int    `form:"offset,default=0"`
	PublicOnly bool `form:"public_only,default=false"`
}

// RecipeSearchResponse represents response for recipe search
type RecipeSearchResponse struct {
	Data       []*RecipeWithIngredients `json:"data"`
	Pagination *Pagination              `json:"pagination"`
}

// AddToMyBookRequest represents request to add a recipe to user's recipe book
type AddToMyBookRequest struct {
	RecipeID int `json:"recipe_id" binding:"required"`
}

// RecipeNutrientSummary represents nutrient summary for a recipe
type RecipeNutrientSummary struct {
	TotalCalories float64 `json:"total_calories"`
	TotalProtein  float64 `json:"total_protein"`
	TotalFat      float64 `json:"total_fat"`
	TotalCarbs    float64 `json:"total_carbs"`
	CaloriesPer100g float64 `json:"calories_per_100g"`
	ProteinPer100g  float64 `json:"protein_per_100g"`
	FatPer100g      float64 `json:"fat_per_100g"`
	CarbsPer100g    float64 `json:"carbs_per_100g"`
}