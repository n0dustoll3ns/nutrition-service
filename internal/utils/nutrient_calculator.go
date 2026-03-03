package utils

import (
	"context"
	"database/sql"
	"fmt"
)

// NutrientCalculator calculates nutritional values for food entries
type NutrientCalculator struct {
	db *sql.DB
}

// NewNutrientCalculator creates a new nutrient calculator
func NewNutrientCalculator(db *sql.DB) *NutrientCalculator {
	return &NutrientCalculator{db: db}
}

// NutrientIDs for common nutrients in USDA database
const (
	NutrientIDEnergy    = 1008 // Energy (calories)
	NutrientIDProtein   = 1003 // Protein
	NutrientIDFat       = 1004 // Total lipid (fat)
	NutrientIDCarbs     = 1005 // Carbohydrate, by difference
)

// CalculatedNutrients represents calculated nutritional values
type CalculatedNutrients struct {
	Calories *float64
	Protein  *float64
	Fat      *float64
	Carbs    *float64
}

// CalculateNutrients calculates nutritional values for a food item
// based on its FDC ID and amount in grams
func (nc *NutrientCalculator) CalculateNutrients(ctx context.Context, fdcID int, amountGrams float64) (*CalculatedNutrients, error) {
	// Query nutrient values per 100g from database
	query := `
		SELECT 
			nutrient_id,
			amount
		FROM nutrition.food_nutrients
		WHERE fdc_id = $1 
			AND nutrient_id IN (1008, 1003, 1004, 1005)
	`

	rows, err := nc.db.QueryContext(ctx, query, fdcID)
	if err != nil {
		return nil, fmt.Errorf("failed to query nutrients for fdc_id %d: %w", fdcID, err)
	}
	defer rows.Close()

	// Map to store nutrient values per 100g
	nutrientsPer100g := make(map[int]float64)
	for rows.Next() {
		var nutrientID int
		var amount float64
		if err := rows.Scan(&nutrientID, &amount); err != nil {
			return nil, fmt.Errorf("failed to scan nutrient for fdc_id %d: %w", fdcID, err)
		}
		nutrientsPer100g[nutrientID] = amount
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating nutrient rows for fdc_id %d: %w", fdcID, err)
	}

	// Calculate values for the specified amount
	result := &CalculatedNutrients{}

	// Calculate calories (Energy)
	if energyPer100g, ok := nutrientsPer100g[NutrientIDEnergy]; ok {
		calories := (energyPer100g / 100.0) * amountGrams
		result.Calories = &calories
	} else {
		// Return zero value instead of nil
		zero := 0.0
		result.Calories = &zero
	}

	// Calculate protein
	if proteinPer100g, ok := nutrientsPer100g[NutrientIDProtein]; ok {
		protein := (proteinPer100g / 100.0) * amountGrams
		result.Protein = &protein
	} else {
		zero := 0.0
		result.Protein = &zero
	}

	// Calculate fat
	if fatPer100g, ok := nutrientsPer100g[NutrientIDFat]; ok {
		fat := (fatPer100g / 100.0) * amountGrams
		result.Fat = &fat
	} else {
		zero := 0.0
		result.Fat = &zero
	}

	// Calculate carbs
	if carbsPer100g, ok := nutrientsPer100g[NutrientIDCarbs]; ok {
		carbs := (carbsPer100g / 100.0) * amountGrams
		result.Carbs = &carbs
	} else {
		zero := 0.0
		result.Carbs = &zero
	}

	return result, nil
}

// CalculateNutrientsWithFallback calculates nutrients with fallback to custom values
// If custom values are provided, they take precedence over calculated values
func (nc *NutrientCalculator) CalculateNutrientsWithFallback(
	ctx context.Context,
	fdcID *int,
	customCalories, customProtein, customFat, customCarbs *float64,
	amountGrams float64,
) (*CalculatedNutrients, error) {
	result := &CalculatedNutrients{}

	// If custom values are provided, use them
	if customCalories != nil {
		result.Calories = customCalories
	}
	if customProtein != nil {
		result.Protein = customProtein
	}
	if customFat != nil {
		result.Fat = customFat
	}
	if customCarbs != nil {
		result.Carbs = customCarbs
	}

	// If we have all custom values, return early
	if result.Calories != nil && result.Protein != nil && result.Fat != nil && result.Carbs != nil {
		return result, nil
	}

	// If FDC ID is provided and we need to calculate some values
	if fdcID != nil {
		calculated, err := nc.CalculateNutrients(ctx, *fdcID, amountGrams)
		if err != nil {
			return nil, fmt.Errorf("failed to calculate nutrients: %w", err)
		}

		// Fill in missing values with calculated ones
		if result.Calories == nil && calculated.Calories != nil {
			result.Calories = calculated.Calories
		}
		if result.Protein == nil && calculated.Protein != nil {
			result.Protein = calculated.Protein
		}
		if result.Fat == nil && calculated.Fat != nil {
			result.Fat = calculated.Fat
		}
		if result.Carbs == nil && calculated.Carbs != nil {
			result.Carbs = calculated.Carbs
		}
	}

	return result, nil
}