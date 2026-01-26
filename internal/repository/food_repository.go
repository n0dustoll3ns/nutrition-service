package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/yourusername/auth-service/internal/model"
	_ "github.com/lib/pq"
)

// FoodRepository defines the interface for food data access
type FoodRepository interface {
	SearchFoods(ctx context.Context, query string, limit, offset int) ([]*model.FoodWithNutrients, int, error)
	GetFoodByID(ctx context.Context, fdcID int) (*model.FoodWithNutrients, error)
	Close() error
}

// foodRepository implements FoodRepository with PostgreSQL
type foodRepository struct {
	db *sql.DB
}

// NewFoodRepository creates a new food repository
func NewFoodRepository(db *sql.DB) FoodRepository {
	return &foodRepository{db: db}
}

// SearchFoods searches for foods by description with pagination
func (r *foodRepository) SearchFoods(ctx context.Context, query string, limit, offset int) ([]*model.FoodWithNutrients, int, error) {
	// First, get total count for pagination
	var total int
	countQuery := `
		SELECT COUNT(*) 
		FROM nutrition.foods 
		WHERE description ILIKE $1
	`
	err := r.db.QueryRowContext(ctx, countQuery, "%"+query+"%").Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count foods: %w", err)
	}

	// Then, get paginated results
	searchQuery := `
		SELECT 
			f.fdc_id, 
			f.description, 
			f.data_type, 
			f.food_class, 
			f.publication_date
		FROM nutrition.foods f
		WHERE f.description ILIKE $1
		ORDER BY 
			CASE 
				WHEN f.description ILIKE $2 THEN 0
				WHEN f.description ILIKE $3 THEN 1
				ELSE 2
			END,
			f.description
		LIMIT $4 OFFSET $5
	`
	
	// Prepare search patterns for relevance sorting
	exactPattern := query + "%"
	containsPattern := "%" + query + "%"
	
	rows, err := r.db.QueryContext(ctx, searchQuery, containsPattern, exactPattern, containsPattern, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to search foods: %w", err)
	}
	defer rows.Close()

	var foods []*model.Food
	for rows.Next() {
		var food model.Food
		err := rows.Scan(
			&food.FDCID,
			&food.Description,
			&food.DataType,
			&food.FoodClass,
			&food.PublicationDate,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan food: %w", err)
		}
		foods = append(foods, &food)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating food rows: %w", err)
	}

	// Get nutrients for each food
	result := make([]*model.FoodWithNutrients, 0, len(foods))
	for _, food := range foods {
		nutrients, err := r.getFoodNutrients(ctx, food.FDCID)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to get nutrients for food %d: %w", food.FDCID, err)
		}
		
		result = append(result, &model.FoodWithNutrients{
			Food:     food,
			Nutrients: nutrients,
		})
	}

	return result, total, nil
}

// getFoodNutrients retrieves nutrients for a specific food
func (r *foodRepository) getFoodNutrients(ctx context.Context, fdcID int) ([]*model.FoodNutrient, error) {
	query := `
		SELECT 
			id, fdc_id, nutrient_id, nutrient_name, nutrient_number,
			unit_name, amount, data_points, min_val, max_val, median,
			derivation_code, derivation_desc
		FROM nutrition.food_nutrients
		WHERE fdc_id = $1
		ORDER BY nutrient_id
	`
	
	rows, err := r.db.QueryContext(ctx, query, fdcID)
	if err != nil {
		return nil, fmt.Errorf("failed to query nutrients: %w", err)
	}
	defer rows.Close()

	var nutrients []*model.FoodNutrient
	for rows.Next() {
		var nutrient model.FoodNutrient
		err := rows.Scan(
			&nutrient.ID,
			&nutrient.FDCID,
			&nutrient.NutrientID,
			&nutrient.NutrientName,
			&nutrient.NutrientNumber,
			&nutrient.UnitName,
			&nutrient.Amount,
			&nutrient.DataPoints,
			&nutrient.MinVal,
			&nutrient.MaxVal,
			&nutrient.Median,
			&nutrient.DerivationCode,
			&nutrient.DerivationDesc,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan nutrient: %w", err)
		}
		nutrients = append(nutrients, &nutrient)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating nutrient rows: %w", err)
	}

	return nutrients, nil
}

// GetFoodByID retrieves a food by its FDC ID with nutrients
func (r *foodRepository) GetFoodByID(ctx context.Context, fdcID int) (*model.FoodWithNutrients, error) {
	// Get food
	query := `
		SELECT 
			f.fdc_id, 
			f.description, 
			f.data_type, 
			f.food_class, 
			f.publication_date
		FROM nutrition.foods f
		WHERE f.fdc_id = $1
	`
	
	var food model.Food
	err := r.db.QueryRowContext(ctx, query, fdcID).Scan(
		&food.FDCID,
		&food.Description,
		&food.DataType,
		&food.FoodClass,
		&food.PublicationDate,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Food not found
		}
		return nil, fmt.Errorf("failed to get food: %w", err)
	}

	// Get nutrients
	nutrients, err := r.getFoodNutrients(ctx, fdcID)
	if err != nil {
		return nil, fmt.Errorf("failed to get nutrients: %w", err)
	}

	return &model.FoodWithNutrients{
		Food:     &food,
		Nutrients: nutrients,
	}, nil
}

// Close closes the database connection
func (r *foodRepository) Close() error {
	return r.db.Close()
}
