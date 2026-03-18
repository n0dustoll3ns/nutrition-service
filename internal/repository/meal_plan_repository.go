package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/n0dustoll3ns/nutrition-service/internal/model"
)

// MealPlanRepository defines the interface for meal plan data access
type MealPlanRepository interface {
	// Basic CRUD
	CreateMealPlan(ctx context.Context, mealPlan *model.MealPlan) (*model.MealPlan, error)
	GetMealPlanByID(ctx context.Context, mealPlanID int) (*model.MealPlanWithEntries, error)
	UpdateMealPlan(ctx context.Context, mealPlanID int, update *model.MealPlanUpdate) error
	DeleteMealPlan(ctx context.Context, mealPlanID int) error
	
	// Listing
	GetUserMealPlans(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*model.MealPlan, int, error)
	
	// Entries
	CreateMealPlanEntry(ctx context.Context, entry *model.MealPlanEntry) error
	GetMealPlanEntries(ctx context.Context, mealPlanID int, limit, offset int) ([]*model.MealPlanEntry, int, error)
	GetMealPlanEntriesByDay(ctx context.Context, mealPlanID int, dayNumber int) ([]*model.MealPlanEntry, error)
	GetMealPlanEntriesByDayAndMeal(ctx context.Context, mealPlanID int, dayNumber int, mealType string) ([]*model.MealPlanEntry, error)
	UpdateMealPlanEntry(ctx context.Context, entryID uuid.UUID, update *model.MealPlanEntryUpdate) error
	DeleteMealPlanEntry(ctx context.Context, entryID uuid.UUID) error
	
	Close() error
}

// mealPlanRepository implements MealPlanRepository with PostgreSQL
type mealPlanRepository struct {
	db *sql.DB
}

// NewMealPlanRepository creates a new meal plan repository
func NewMealPlanRepository(db *sql.DB) MealPlanRepository {
	return &mealPlanRepository{db: db}
}

// CreateMealPlan creates a new meal plan
func (r *mealPlanRepository) CreateMealPlan(ctx context.Context, mealPlan *model.MealPlan) (*model.MealPlan, error) {
	query := `
		INSERT INTO meal_plan.meal_plans (
			user_id, name, description, days_count, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`

	var mealPlanID int
	err := r.db.QueryRowContext(ctx, query,
		mealPlan.UserID,
		mealPlan.Name,
		mealPlan.Description,
		mealPlan.DaysCount,
		mealPlan.CreatedAt,
		mealPlan.UpdatedAt,
	).Scan(&mealPlanID)
	if err != nil {
		return nil, fmt.Errorf("failed to create meal plan: %w", err)
	}

	mealPlan.ID = mealPlanID
	return mealPlan, nil
}

// GetMealPlanByID retrieves a meal plan by its ID with entries
func (r *mealPlanRepository) GetMealPlanByID(ctx context.Context, mealPlanID int) (*model.MealPlanWithEntries, error) {
	// Get meal plan
	mealPlanQuery := `
		SELECT 
			id, user_id, name, description, days_count, created_at, updated_at
		FROM meal_plan.meal_plans
		WHERE id = $1
	`

	var mealPlan model.MealPlan
	err := r.db.QueryRowContext(ctx, mealPlanQuery, mealPlanID).Scan(
		&mealPlan.ID,
		&mealPlan.UserID,
		&mealPlan.Name,
		&mealPlan.Description,
		&mealPlan.DaysCount,
		&mealPlan.CreatedAt,
		&mealPlan.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Meal plan not found
		}
		return nil, fmt.Errorf("failed to get meal plan: %w", err)
	}

	// Get entries
	entries, err := r.getMealPlanEntries(ctx, mealPlanID)
	if err != nil {
		return nil, fmt.Errorf("failed to get meal plan entries: %w", err)
	}

	return &model.MealPlanWithEntries{
		MealPlan: &mealPlan,
		Entries:  entries,
	}, nil
}

// getMealPlanEntries retrieves entries for a specific meal plan
func (r *mealPlanRepository) getMealPlanEntries(ctx context.Context, mealPlanID int) ([]*model.MealPlanEntry, error) {
	query := `
		SELECT 
			id, meal_plan_id, day_number, meal_type, fdc_id, custom_food_name, recipe_id, amount_grams
		FROM meal_plan.meal_plan_entries
		WHERE meal_plan_id = $1
		ORDER BY day_number, 
			CASE meal_type
				WHEN 'breakfast' THEN 1
				WHEN 'brunch' THEN 2
				WHEN 'lunch' THEN 3
				WHEN 'afternoon_snack' THEN 4
				WHEN 'dinner' THEN 5
				WHEN 'snack' THEN 6
			END
	`

	rows, err := r.db.QueryContext(ctx, query, mealPlanID)
	if err != nil {
		return nil, fmt.Errorf("failed to query meal plan entries: %w", err)
	}
	defer rows.Close()

	var entries []*model.MealPlanEntry
	for rows.Next() {
		var entry model.MealPlanEntry
		err := rows.Scan(
			&entry.ID,
			&entry.MealPlanID,
			&entry.DayNumber,
			&entry.MealType,
			&entry.FDCID,
			&entry.CustomFoodName,
			&entry.RecipeID,
			&entry.AmountGrams,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan meal plan entry: %w", err)
		}
		entries = append(entries, &entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating meal plan entry rows: %w", err)
	}

	return entries, nil
}

// UpdateMealPlan updates a meal plan
func (r *mealPlanRepository) UpdateMealPlan(ctx context.Context, mealPlanID int, update *model.MealPlanUpdate) error {
	query := "UPDATE meal_plan.meal_plans SET "
	args := []interface{}{}
	argIndex := 1
	
	if update.Name != nil {
		query += fmt.Sprintf("name = $%d, ", argIndex)
		args = append(args, *update.Name)
		argIndex++
	}
	
	if update.Description != nil {
		query += fmt.Sprintf("description = $%d, ", argIndex)
		args = append(args, *update.Description)
		argIndex++
	}
	
	if len(args) == 0 {
		return nil // Nothing to update
	}
	
	query = query[:len(query)-2] // Remove ", "
	query += fmt.Sprintf(" WHERE id = $%d", argIndex)
	args = append(args, mealPlanID)
	
	_, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update meal plan: %w", err)
	}
	
	return nil
}

// DeleteMealPlan deletes a meal plan by ID
func (r *mealPlanRepository) DeleteMealPlan(ctx context.Context, mealPlanID int) error {
	query := "DELETE FROM meal_plan.meal_plans WHERE id = $1"
	
	_, err := r.db.ExecContext(ctx, query, mealPlanID)
	if err != nil {
		return fmt.Errorf("failed to delete meal plan: %w", err)
	}
	
	return nil
}

// GetUserMealPlans retrieves meal plans created by a user
func (r *mealPlanRepository) GetUserMealPlans(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*model.MealPlan, int, error) {
	// Get total count
	var total int
	countQuery := `SELECT COUNT(*) FROM meal_plan.meal_plans WHERE user_id = $1`
	err := r.db.QueryRowContext(ctx, countQuery, userID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count user meal plans: %w", err)
	}

	// Get paginated meal plans
	query := `
		SELECT 
			id, user_id, name, description, days_count, created_at, updated_at
		FROM meal_plan.meal_plans
		WHERE user_id = $1
		ORDER BY updated_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query user meal plans: %w", err)
	}
	defer rows.Close()

	var mealPlans []*model.MealPlan
	for rows.Next() {
		var mealPlan model.MealPlan
		err := rows.Scan(
			&mealPlan.ID,
			&mealPlan.UserID,
			&mealPlan.Name,
			&mealPlan.Description,
			&mealPlan.DaysCount,
			&mealPlan.CreatedAt,
			&mealPlan.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan meal plan: %w", err)
		}
		mealPlans = append(mealPlans, &mealPlan)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating meal plan rows: %w", err)
	}

	return mealPlans, total, nil
}

// CreateMealPlanEntry creates a new meal plan entry
func (r *mealPlanRepository) CreateMealPlanEntry(ctx context.Context, entry *model.MealPlanEntry) error {
	query := `
		INSERT INTO meal_plan.meal_plan_entries (
			id, meal_plan_id, day_number, meal_type, fdc_id, custom_food_name, recipe_id, amount_grams
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	
	_, err := r.db.ExecContext(ctx, query,
		entry.ID,
		entry.MealPlanID,
		entry.DayNumber,
		entry.MealType,
		entry.FDCID,
		entry.CustomFoodName,
		entry.RecipeID,
		entry.AmountGrams,
	)
	
	if err != nil {
		return fmt.Errorf("failed to create meal plan entry: %w", err)
	}
	
	return nil
}

// GetMealPlanEntries retrieves entries for a meal plan with pagination
func (r *mealPlanRepository) GetMealPlanEntries(ctx context.Context, mealPlanID int, limit, offset int) ([]*model.MealPlanEntry, int, error) {
	// Get total count
	var total int
	countQuery := `SELECT COUNT(*) FROM meal_plan.meal_plan_entries WHERE meal_plan_id = $1`
	err := r.db.QueryRowContext(ctx, countQuery, mealPlanID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count meal plan entries: %w", err)
	}

	// Get paginated entries
	query := `
		SELECT 
			id, meal_plan_id, day_number, meal_type, fdc_id, custom_food_name, recipe_id, amount_grams
		FROM meal_plan.meal_plan_entries
		WHERE meal_plan_id = $1
		ORDER BY day_number, 
			CASE meal_type
				WHEN 'breakfast' THEN 1
				WHEN 'brunch' THEN 2
				WHEN 'lunch' THEN 3
				WHEN 'afternoon_snack' THEN 4
				WHEN 'dinner' THEN 5
				WHEN 'snack' THEN 6
			END
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, mealPlanID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query meal plan entries: %w", err)
	}
	defer rows.Close()

	var entries []*model.MealPlanEntry
	for rows.Next() {
		var entry model.MealPlanEntry
		err := rows.Scan(
			&entry.ID,
			&entry.MealPlanID,
			&entry.DayNumber,
			&entry.MealType,
			&entry.FDCID,
			&entry.CustomFoodName,
			&entry.RecipeID,
			&entry.AmountGrams,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan meal plan entry: %w", err)
		}
		entries = append(entries, &entry)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating meal plan entry rows: %w", err)
	}

	return entries, total, nil
}

// GetMealPlanEntriesByDay retrieves entries for a specific day in a meal plan
func (r *mealPlanRepository) GetMealPlanEntriesByDay(ctx context.Context, mealPlanID int, dayNumber int) ([]*model.MealPlanEntry, error) {
	query := `
		SELECT 
			id, meal_plan_id, day_number, meal_type, fdc_id, custom_food_name, recipe_id, amount_grams
		FROM meal_plan.meal_plan_entries
		WHERE meal_plan_id = $1 AND day_number = $2
		ORDER BY 
			CASE meal_type
				WHEN 'breakfast' THEN 1
				WHEN 'brunch' THEN 2
				WHEN 'lunch' THEN 3
				WHEN 'afternoon_snack' THEN 4
				WHEN 'dinner' THEN 5
				WHEN 'snack' THEN 6
			END
	`

	rows, err := r.db.QueryContext(ctx, query, mealPlanID, dayNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to query meal plan entries by day: %w", err)
	}
	defer rows.Close()

	var entries []*model.MealPlanEntry
	for rows.Next() {
		var entry model.MealPlanEntry
		err := rows.Scan(
			&entry.ID,
			&entry.MealPlanID,
			&entry.DayNumber,
			&entry.MealType,
			&entry.FDCID,
			&entry.CustomFoodName,
			&entry.RecipeID,
			&entry.AmountGrams,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan meal plan entry: %w", err)
		}
		entries = append(entries, &entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating meal plan entry rows: %w", err)
	}

	return entries, nil
}

// GetMealPlanEntriesByDayAndMeal retrieves entries for a specific day and meal type in a meal plan
func (r *mealPlanRepository) GetMealPlanEntriesByDayAndMeal(ctx context.Context, mealPlanID int, dayNumber int, mealType string) ([]*model.MealPlanEntry, error) {
	query := `
		SELECT 
			id, meal_plan_id, day_number, meal_type, fdc_id, custom_food_name, recipe_id, amount_grams
		FROM meal_plan.meal_plan_entries
		WHERE meal_plan_id = $1 AND day_number = $2 AND meal_type = $3
	`

	rows, err := r.db.QueryContext(ctx, query, mealPlanID, dayNumber, mealType)
	if err != nil {
		return nil, fmt.Errorf("failed to query meal plan entries by day and meal: %w", err)
	}
	defer rows.Close()

	var entries []*model.MealPlanEntry
	for rows.Next() {
		var entry model.MealPlanEntry
		err := rows.Scan(
			&entry.ID,
			&entry.MealPlanID,
			&entry.DayNumber,
			&entry.MealType,
			&entry.FDCID,
			&entry.CustomFoodName,
			&entry.RecipeID,
			&entry.AmountGrams,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan meal plan entry: %w", err)
		}
		entries = append(entries, &entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating meal plan entry rows: %w", err)
	}

	return entries, nil
}

// UpdateMealPlanEntry updates a meal plan entry
func (r *mealPlanRepository) UpdateMealPlanEntry(ctx context.Context, entryID uuid.UUID, update *model.MealPlanEntryUpdate) error {
	query := "UPDATE meal_plan.meal_plan_entries SET "
	args := []interface{}{}
	argIndex := 1
	
	if update.DayNumber != nil {
		query += fmt.Sprintf("day_number = $%d, ", argIndex)
		args = append(args, *update.DayNumber)
		argIndex++
	}
	
	if update.MealType != nil {
		query += fmt.Sprintf("meal_type = $%d, ", argIndex)
		args = append(args, *update.MealType)
		argIndex++
	}
	
	if update.AmountGrams != nil {
		query += fmt.Sprintf("amount_grams = $%d, ", argIndex)
		args = append(args, *update.AmountGrams)
		argIndex++
	}
	
	if len(args) == 0 {
		return nil // Nothing to update
	}
	
	query = query[:len(query)-2] // Remove ", "
	query += fmt.Sprintf(" WHERE id = $%d", argIndex)
	args = append(args, entryID)
	
	_, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update meal plan entry: %w", err)
	}
	
	return nil
}

// DeleteMealPlanEntry deletes a meal plan entry by ID
func (r *mealPlanRepository) DeleteMealPlanEntry(ctx context.Context, entryID uuid.UUID) error {
	query := "DELETE FROM meal_plan.meal_plan_entries WHERE id = $1"
	
	_, err := r.db.ExecContext(ctx, query, entryID)
	if err != nil {
		return fmt.Errorf("failed to delete meal plan entry: %w", err)
	}
	
	return nil
}

// Close closes the database connection
func (r *mealPlanRepository) Close() error {
	return r.db.Close()
}
