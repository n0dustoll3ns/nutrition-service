package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/yourusername/auth-service/internal/model"
)

// DiaryRepository defines the interface for diary data access
type DiaryRepository interface {
	// Food entries
	CreateFoodEntry(ctx context.Context, entry *model.FoodEntry) error
	GetFoodEntryByID(ctx context.Context, id uuid.UUID) (*model.FoodEntry, error)
	GetFoodEntriesByPeriod(ctx context.Context, userID uuid.UUID, startDate, endDate time.Time) ([]*model.FoodEntry, error)
	UpdateFoodEntry(ctx context.Context, id uuid.UUID, update *model.FoodEntryUpdate) error
	DeleteFoodEntry(ctx context.Context, id uuid.UUID) error
	DeleteFoodEntriesByUserAndDate(ctx context.Context, userID uuid.UUID, date time.Time) error
	
	// Statistics
	GetDaySummary(ctx context.Context, userID uuid.UUID, date time.Time) (*model.DaySummary, error)
	GetPeriodSummary(ctx context.Context, userID uuid.UUID, startDate, endDate time.Time) (*model.DaySummary, error)
	
	// Copy
	CopyFoodEntries(ctx context.Context, userID uuid.UUID, sourceDate, targetDate time.Time) error
	
	Close() error
}

// diaryRepository implements DiaryRepository with PostgreSQL
type diaryRepository struct {
	db *sql.DB
}

// NewDiaryRepository creates a new diary repository
func NewDiaryRepository(db *sql.DB) DiaryRepository {
	return &diaryRepository{db: db}
}

// CreateFoodEntry creates a new food entry in the diary
func (r *diaryRepository) CreateFoodEntry(ctx context.Context, entry *model.FoodEntry) error {
	query := `
		INSERT INTO diary.food_entries (
			id, user_id, date, meal_type, fdc_id, custom_food_name,
			amount_grams, calculated_calories, calculated_protein,
			calculated_fat, calculated_carbs, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`
	
	_, err := r.db.ExecContext(ctx, query,
		entry.ID,
		entry.UserID,
		entry.Date,
		entry.MealType,
		entry.FDCID,
		entry.CustomFoodName,
		entry.AmountGrams,
		entry.CalculatedCalories,
		entry.CalculatedProtein,
		entry.CalculatedFat,
		entry.CalculatedCarbs,
		entry.CreatedAt,
	)
	
	if err != nil {
		return fmt.Errorf("failed to create food entry: %w", err)
	}
	
	return nil
}

// GetFoodEntryByID retrieves a food entry by its ID
func (r *diaryRepository) GetFoodEntryByID(ctx context.Context, id uuid.UUID) (*model.FoodEntry, error) {
	query := `
		SELECT 
			id, user_id, date, meal_type, fdc_id, custom_food_name,
			amount_grams, calculated_calories, calculated_protein,
			calculated_fat, calculated_carbs, created_at
		FROM diary.food_entries
		WHERE id = $1
	`
	
	var entry model.FoodEntry
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&entry.ID,
		&entry.UserID,
		&entry.Date,
		&entry.MealType,
		&entry.FDCID,
		&entry.CustomFoodName,
		&entry.AmountGrams,
		&entry.CalculatedCalories,
		&entry.CalculatedProtein,
		&entry.CalculatedFat,
		&entry.CalculatedCarbs,
		&entry.CreatedAt,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Entry not found
		}
		return nil, fmt.Errorf("failed to get food entry: %w", err)
	}
	
	return &entry, nil
}

// GetFoodEntriesByPeriod retrieves food entries for a user within a date period
func (r *diaryRepository) GetFoodEntriesByPeriod(ctx context.Context, userID uuid.UUID, startDate, endDate time.Time) ([]*model.FoodEntry, error) {
	query := `
		SELECT 
			id, user_id, date, meal_type, fdc_id, custom_food_name,
			amount_grams, calculated_calories, calculated_protein,
			calculated_fat, calculated_carbs, created_at
		FROM diary.food_entries
		WHERE user_id = $1 AND date >= $2 AND date <= $3
		ORDER BY date DESC, 
			CASE meal_type
				WHEN 'breakfast' THEN 1
				WHEN 'brunch' THEN 2
				WHEN 'lunch' THEN 3
				WHEN 'afternoon_snack' THEN 4
				WHEN 'dinner' THEN 5
				WHEN 'snack' THEN 6
			END,
			created_at
	`
	
	rows, err := r.db.QueryContext(ctx, query, userID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to query food entries: %w", err)
	}
	defer rows.Close()
	
	var entries []*model.FoodEntry
	for rows.Next() {
		var entry model.FoodEntry
		err := rows.Scan(
			&entry.ID,
			&entry.UserID,
			&entry.Date,
			&entry.MealType,
			&entry.FDCID,
			&entry.CustomFoodName,
			&entry.AmountGrams,
			&entry.CalculatedCalories,
			&entry.CalculatedProtein,
			&entry.CalculatedFat,
			&entry.CalculatedCarbs,
			&entry.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan food entry: %w", err)
		}
		entries = append(entries, &entry)
	}
	
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating food entry rows: %w", err)
	}
	
	return entries, nil
}

// UpdateFoodEntry updates a food entry
func (r *diaryRepository) UpdateFoodEntry(ctx context.Context, id uuid.UUID, update *model.FoodEntryUpdate) error {
	// Build dynamic query based on provided fields
	query := "UPDATE diary.food_entries SET "
	args := []interface{}{}
	argIndex := 1
	
	if update.AmountGrams != nil {
		query += fmt.Sprintf("amount_grams = $%d, ", argIndex)
		args = append(args, *update.AmountGrams)
		argIndex++
	}
	
	if update.CustomCalories != nil {
		query += fmt.Sprintf("calculated_calories = $%d, ", argIndex)
		args = append(args, *update.CustomCalories)
		argIndex++
	}
	
	if update.CustomProtein != nil {
		query += fmt.Sprintf("calculated_protein = $%d, ", argIndex)
		args = append(args, *update.CustomProtein)
		argIndex++
	}
	
	if update.CustomFat != nil {
		query += fmt.Sprintf("calculated_fat = $%d, ", argIndex)
		args = append(args, *update.CustomFat)
		argIndex++
	}
	
	if update.CustomCarbs != nil {
		query += fmt.Sprintf("calculated_carbs = $%d, ", argIndex)
		args = append(args, *update.CustomCarbs)
		argIndex++
	}
	
	// Remove trailing comma and space
	if len(args) == 0 {
		return nil // Nothing to update
	}
	
	query = query[:len(query)-2] // Remove ", "
	query += fmt.Sprintf(" WHERE id = $%d", argIndex)
	args = append(args, id)
	
	_, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update food entry: %w", err)
	}
	
	return nil
}

// DeleteFoodEntry deletes a food entry by ID
func (r *diaryRepository) DeleteFoodEntry(ctx context.Context, id uuid.UUID) error {
	query := "DELETE FROM diary.food_entries WHERE id = $1"
	
	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete food entry: %w", err)
	}
	
	return nil
}

// DeleteFoodEntriesByUserAndDate deletes all food entries for a user on a specific date
func (r *diaryRepository) DeleteFoodEntriesByUserAndDate(ctx context.Context, userID uuid.UUID, date time.Time) error {
	query := "DELETE FROM diary.food_entries WHERE user_id = $1 AND date = $2"
	
	_, err := r.db.ExecContext(ctx, query, userID, date)
	if err != nil {
		return fmt.Errorf("failed to delete food entries: %w", err)
	}
	
	return nil
}

// GetDaySummary calculates nutritional summary for a specific day
func (r *diaryRepository) GetDaySummary(ctx context.Context, userID uuid.UUID, date time.Time) (*model.DaySummary, error) {
	query := `
		SELECT 
			COALESCE(SUM(calculated_calories), 0) as total_calories,
			COALESCE(SUM(calculated_protein), 0) as total_protein,
			COALESCE(SUM(calculated_fat), 0) as total_fat,
			COALESCE(SUM(calculated_carbs), 0) as total_carbs,
			COUNT(DISTINCT meal_type) as meal_count,
			COUNT(*) as food_count
		FROM diary.food_entries
		WHERE user_id = $1 AND date = $2
	`
	
	var summary model.DaySummary
	err := r.db.QueryRowContext(ctx, query, userID, date).Scan(
		&summary.TotalCalories,
		&summary.TotalProtein,
		&summary.TotalFat,
		&summary.TotalCarbs,
		&summary.MealCount,
		&summary.FoodCount,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			// Return empty summary if no entries
			return &model.DaySummary{}, nil
		}
		return nil, fmt.Errorf("failed to get day summary: %w", err)
	}
	
	return &summary, nil
}

// GetPeriodSummary calculates nutritional summary for a date period
func (r *diaryRepository) GetPeriodSummary(ctx context.Context, userID uuid.UUID, startDate, endDate time.Time) (*model.DaySummary, error) {
	query := `
		SELECT 
			COALESCE(SUM(calculated_calories), 0) as total_calories,
			COALESCE(SUM(calculated_protein), 0) as total_protein,
			COALESCE(SUM(calculated_fat), 0) as total_fat,
			COALESCE(SUM(calculated_carbs), 0) as total_carbs,
			COUNT(DISTINCT date || meal_type) as meal_count,
			COUNT(*) as food_count
		FROM diary.food_entries
		WHERE user_id = $1 AND date >= $2 AND date <= $3
	`
	
	var summary model.DaySummary
	err := r.db.QueryRowContext(ctx, query, userID, startDate, endDate).Scan(
		&summary.TotalCalories,
		&summary.TotalProtein,
		&summary.TotalFat,
		&summary.TotalCarbs,
		&summary.MealCount,
		&summary.FoodCount,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			// Return empty summary if no entries
			return &model.DaySummary{}, nil
		}
		return nil, fmt.Errorf("failed to get period summary: %w", err)
	}
	
	return &summary, nil
}

// CopyFoodEntries copies food entries from one date to another for a user
func (r *diaryRepository) CopyFoodEntries(ctx context.Context, userID uuid.UUID, sourceDate, targetDate time.Time) error {
	// Use transaction to ensure atomicity
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()
	
	// First, delete existing entries on target date
	deleteQuery := "DELETE FROM diary.food_entries WHERE user_id = $1 AND date = $2"
	_, err = tx.ExecContext(ctx, deleteQuery, userID, targetDate)
	if err != nil {
		return fmt.Errorf("failed to delete existing entries: %w", err)
	}
	
	// Copy entries from source date to target date
	copyQuery := `
		INSERT INTO diary.food_entries (
			id, user_id, date, meal_type, fdc_id, custom_food_name,
			amount_grams, calculated_calories, calculated_protein,
			calculated_fat, calculated_carbs, created_at
		)
		SELECT 
			gen_random_uuid(), user_id, $3, meal_type, fdc_id, custom_food_name,
			amount_grams, calculated_calories, calculated_protein,
			calculated_fat, calculated_carbs, NOW()
		FROM diary.food_entries
		WHERE user_id = $1 AND date = $2
	`
	
	_, err = tx.ExecContext(ctx, copyQuery, userID, sourceDate, targetDate)
	if err != nil {
		return fmt.Errorf("failed to copy food entries: %w", err)
	}
	
	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	
	return nil
}

// Close closes the database connection
func (r *diaryRepository) Close() error {
	return r.db.Close()
}