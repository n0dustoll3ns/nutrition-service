package model

import "time"

// Food represents a food product from the USDA database
type Food struct {
	FDCID          int       `json:"fdc_id"`
	Description    string    `json:"description"`
	DataType       string    `json:"data_type"`
	FoodClass      string    `json:"food_class"`
	PublicationDate string   `json:"publication_date"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// FoodNutrient represents a nutrient value for a food
type FoodNutrient struct {
	ID             int     `json:"id"`
	FDCID          int     `json:"fdc_id"`
	NutrientID     int     `json:"nutrient_id"`
	NutrientName   string  `json:"nutrient_name"`
	NutrientNumber string  `json:"nutrient_number"`
	UnitName       string  `json:"unit_name"`
	Amount         float64 `json:"amount"`
	DataPoints     int     `json:"data_points"`
	MinVal         float64 `json:"min_val"`
	MaxVal         float64 `json:"max_val"`
	Median         float64 `json:"median"`
	DerivationCode string  `json:"derivation_code"`
	DerivationDesc string  `json:"derivation_desc"`
}

// FoodWithNutrients represents a food with its associated nutrients
type FoodWithNutrients struct {
	Food     *Food           `json:"food"`
	Nutrients []*FoodNutrient `json:"nutrients"`
}

// SearchFoodRequest represents the request parameters for searching foods
type SearchFoodRequest struct {
	Query  string `form:"q" binding:"required"`
	Limit  int    `form:"limit,default=20"`
	Offset int    `form:"offset,default=0"`
}

// SearchFoodResponse represents the response for food search
type SearchFoodResponse struct {
	Data       []*FoodWithNutrients `json:"data"`
	Pagination *Pagination          `json:"pagination"`
}

// Pagination represents pagination metadata
type Pagination struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}