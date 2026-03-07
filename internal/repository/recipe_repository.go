package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/yourusername/auth-service/internal/model"
)

// RecipeRepository defines the interface for recipe data access
type RecipeRepository interface {
	// Basic CRUD
	CreateRecipe(ctx context.Context, recipe *model.Recipe, ingredients []*model.RecipeIngredient) (*model.RecipeWithIngredients, error)
	GetRecipeByID(ctx context.Context, recipeID int) (*model.RecipeWithIngredients, error)
	UpdateRecipe(ctx context.Context, recipeID int, update *model.RecipeUpdate) error
	DeleteRecipe(ctx context.Context, recipeID int) error
	
	// Listing
	GetUserRecipes(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*model.RecipeWithIngredients, int, error)
	GetPublicRecipes(ctx context.Context, limit, offset int) ([]*model.RecipeWithIngredients, int, error)
	SearchRecipes(ctx context.Context, query string, publicOnly bool, limit, offset int) ([]*model.RecipeWithIngredients, int, error)
	
	// Recipe book
	AddRecipeToUserBook(ctx context.Context, userID uuid.UUID, recipeID int) error
	GetUserRecipeBook(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*model.RecipeWithIngredients, int, error)
	
	Close() error
}

// recipeRepository implements RecipeRepository with PostgreSQL
type recipeRepository struct {
	db *sql.DB
}

// NewRecipeRepository creates a new recipe repository
func NewRecipeRepository(db *sql.DB) RecipeRepository {
	return &recipeRepository{db: db}
}

// CreateRecipe creates a new recipe with ingredients
func (r *recipeRepository) CreateRecipe(ctx context.Context, recipe *model.Recipe, ingredients []*model.RecipeIngredient) (*model.RecipeWithIngredients, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Insert recipe
	recipeQuery := `
		INSERT INTO recipe.recipes (
			user_id, name, description, category, is_public, total_weight,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`

	var recipeID int
	err = tx.QueryRowContext(ctx, recipeQuery,
		recipe.UserID,
		recipe.Name,
		recipe.Description,
		recipe.Category,
		recipe.IsPublic,
		recipe.TotalWeight,
		recipe.CreatedAt,
		recipe.UpdatedAt,
	).Scan(&recipeID)
	if err != nil {
		return nil, fmt.Errorf("failed to create recipe: %w", err)
	}

	// Insert ingredients
	ingredientQuery := `
		INSERT INTO recipe.recipe_ingredients (
			id, recipe_id, fdc_id, custom_food_name, amount_grams,
			calories_per_100g, protein_per_100g, fat_per_100g, carbs_per_100g
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	for _, ingredient := range ingredients {
		ingredient.ID = uuid.New()
		ingredient.RecipeID = recipeID
		
		_, err := tx.ExecContext(ctx, ingredientQuery,
			ingredient.ID,
			ingredient.RecipeID,
			ingredient.FDCID,
			ingredient.CustomFoodName,
			ingredient.AmountGrams,
			ingredient.CaloriesPer100g,
			ingredient.ProteinPer100g,
			ingredient.FatPer100g,
			ingredient.CarbsPer100g,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create recipe ingredient: %w", err)
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Get created recipe with ingredients
	return r.GetRecipeByID(ctx, recipeID)
}

// GetRecipeByID retrieves a recipe by its ID with ingredients
func (r *recipeRepository) GetRecipeByID(ctx context.Context, recipeID int) (*model.RecipeWithIngredients, error) {
	// Get recipe
	recipeQuery := `
		SELECT 
			id, user_id, name, description, category, is_public,
			total_weight, created_at, updated_at
		FROM recipe.recipes
		WHERE id = $1
	`

	var recipe model.Recipe
	err := r.db.QueryRowContext(ctx, recipeQuery, recipeID).Scan(
		&recipe.ID,
		&recipe.UserID,
		&recipe.Name,
		&recipe.Description,
		&recipe.Category,
		&recipe.IsPublic,
		&recipe.TotalWeight,
		&recipe.CreatedAt,
		&recipe.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Recipe not found
		}
		return nil, fmt.Errorf("failed to get recipe: %w", err)
	}

	// Get ingredients
	ingredients, err := r.getRecipeIngredients(ctx, recipeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get recipe ingredients: %w", err)
	}

	return &model.RecipeWithIngredients{
		Recipe:      &recipe,
		Ingredients: ingredients,
	}, nil
}

// getRecipeIngredients retrieves ingredients for a specific recipe
func (r *recipeRepository) getRecipeIngredients(ctx context.Context, recipeID int) ([]*model.RecipeIngredient, error) {
	query := `
		SELECT 
			id, recipe_id, fdc_id, custom_food_name, amount_grams,
			calories_per_100g, protein_per_100g, fat_per_100g, carbs_per_100g
		FROM recipe.recipe_ingredients
		WHERE recipe_id = $1
		ORDER BY created_at
	`

	rows, err := r.db.QueryContext(ctx, query, recipeID)
	if err != nil {
		return nil, fmt.Errorf("failed to query recipe ingredients: %w", err)
	}
	defer rows.Close()

	var ingredients []*model.RecipeIngredient
	for rows.Next() {
		var ingredient model.RecipeIngredient
		err := rows.Scan(
			&ingredient.ID,
			&ingredient.RecipeID,
			&ingredient.FDCID,
			&ingredient.CustomFoodName,
			&ingredient.AmountGrams,
			&ingredient.CaloriesPer100g,
			&ingredient.ProteinPer100g,
			&ingredient.FatPer100g,
			&ingredient.CarbsPer100g,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan recipe ingredient: %w", err)
		}
		ingredients = append(ingredients, &ingredient)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating recipe ingredient rows: %w", err)
	}

	return ingredients, nil
}

// UpdateRecipe updates a recipe
func (r *recipeRepository) UpdateRecipe(ctx context.Context, recipeID int, update *model.RecipeUpdate) error {
	query := "UPDATE recipe.recipes SET "
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
	
	if update.Category != nil {
		query += fmt.Sprintf("category = $%d, ", argIndex)
		args = append(args, *update.Category)
		argIndex++
	}
	
	if update.IsPublic != nil {
		query += fmt.Sprintf("is_public = $%d, ", argIndex)
		args = append(args, *update.IsPublic)
		argIndex++
	}
	
	if len(args) == 0 {
		return nil // Nothing to update
	}
	
	query = query[:len(query)-2] // Remove ", "
	query += fmt.Sprintf(" WHERE id = $%d", argIndex)
	args = append(args, recipeID)
	
	_, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update recipe: %w", err)
	}
	
	return nil
}

// DeleteRecipe deletes a recipe by ID
func (r *recipeRepository) DeleteRecipe(ctx context.Context, recipeID int) error {
	query := "DELETE FROM recipe.recipes WHERE id = $1"
	
	_, err := r.db.ExecContext(ctx, query, recipeID)
	if err != nil {
		return fmt.Errorf("failed to delete recipe: %w", err)
	}
	
	return nil
}

// GetUserRecipes retrieves recipes created by a user
func (r *recipeRepository) GetUserRecipes(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*model.RecipeWithIngredients, int, error) {
	// Get total count
	var total int
	countQuery := `SELECT COUNT(*) FROM recipe.recipes WHERE user_id = $1`
	err := r.db.QueryRowContext(ctx, countQuery, userID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count user recipes: %w", err)
	}

	// Get paginated recipes
	query := `
		SELECT 
			id, user_id, name, description, category, is_public,
			total_weight, created_at, updated_at
		FROM recipe.recipes
		WHERE user_id = $1
		ORDER BY updated_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query user recipes: %w", err)
	}
	defer rows.Close()

	var recipes []*model.Recipe
	for rows.Next() {
		var recipe model.Recipe
		err := rows.Scan(
			&recipe.ID,
			&recipe.UserID,
			&recipe.Name,
			&recipe.Description,
			&recipe.Category,
			&recipe.IsPublic,
			&recipe.TotalWeight,
			&recipe.CreatedAt,
			&recipe.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan recipe: %w", err)
		}
		recipes = append(recipes, &recipe)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating recipe rows: %w", err)
	}

	// Get ingredients for each recipe
	result := make([]*model.RecipeWithIngredients, 0, len(recipes))
	for _, recipe := range recipes {
		ingredients, err := r.getRecipeIngredients(ctx, recipe.ID)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to get ingredients for recipe %d: %w", recipe.ID, err)
		}

		result = append(result, &model.RecipeWithIngredients{
			Recipe:      recipe,
			Ingredients: ingredients,
		})
	}

	return result, total, nil
}

// GetPublicRecipes retrieves public recipes
func (r *recipeRepository) GetPublicRecipes(ctx context.Context, limit, offset int) ([]*model.RecipeWithIngredients, int, error) {
	// Get total count
	var total int
	countQuery := `SELECT COUNT(*) FROM recipe.recipes WHERE is_public = true`
	err := r.db.QueryRowContext(ctx, countQuery).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count public recipes: %w", err)
	}

	// Get paginated recipes
	query := `
		SELECT 
			id, user_id, name, description, category, is_public,
			total_weight, created_at, updated_at
		FROM recipe.recipes
		WHERE is_public = true
		ORDER BY updated_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query public recipes: %w", err)
	}
	defer rows.Close()

	var recipes []*model.Recipe
	for rows.Next() {
		var recipe model.Recipe
		err := rows.Scan(
			&recipe.ID,
			&recipe.UserID,
			&recipe.Name,
			&recipe.Description,
			&recipe.Category,
			&recipe.IsPublic,
			&recipe.TotalWeight,
			&recipe.CreatedAt,
			&recipe.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan recipe: %w", err)
		}
		recipes = append(recipes, &recipe)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating recipe rows: %w", err)
	}

	// Get ingredients for each recipe
	result := make([]*model.RecipeWithIngredients, 0, len(recipes))
	for _, recipe := range recipes {
		ingredients, err := r.getRecipeIngredients(ctx, recipe.ID)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to get ingredients for recipe %d: %w", recipe.ID, err)
		}

		result = append(result, &model.RecipeWithIngredients{
			Recipe:      recipe,
			Ingredients: ingredients,
		})
	}

	return result, total, nil
}

// SearchRecipes searches for recipes by name
func (r *recipeRepository) SearchRecipes(ctx context.Context, query string, publicOnly bool, limit, offset int) ([]*model.RecipeWithIngredients, int, error) {
	// Build count query
	countQuery := `SELECT COUNT(*) FROM recipe.recipes WHERE name ILIKE $1`
	countArgs := []interface{}{"%" + query + "%"}
	
	if publicOnly {
		countQuery += " AND is_public = true"
	}

	var total int
	err := r.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count recipes: %w", err)
	}

	// Build search query
	searchQuery := `SELECT id, user_id, name, description, category, is_public, total_weight, created_at, updated_at FROM recipe.recipes WHERE name ILIKE $1`
	searchArgs := []interface{}{"%" + query + "%"}
	
	if publicOnly {
		searchQuery += " AND is_public = true"
	}
	
	searchQuery += " ORDER BY name LIMIT $2 OFFSET $3"
	searchArgs = append(searchArgs, limit, offset)

	rows, err := r.db.QueryContext(ctx, searchQuery, searchArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to search recipes: %w", err)
	}
	defer rows.Close()

	var recipes []*model.Recipe
	for rows.Next() {
		var recipe model.Recipe
		err := rows.Scan(
			&recipe.ID,
			&recipe.UserID,
			&recipe.Name,
			&recipe.Description,
			&recipe.Category,
			&recipe.IsPublic,
			&recipe.TotalWeight,
			&recipe.CreatedAt,
			&recipe.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan recipe: %w", err)
		}
		recipes = append(recipes, &recipe)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating recipe rows: %w", err)
	}

	// Get ingredients for each recipe
	result := make([]*model.RecipeWithIngredients, 0, len(recipes))
	for _, recipe := range recipes {
		ingredients, err := r.getRecipeIngredients(ctx, recipe.ID)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to get ingredients for recipe %d: %w", recipe.ID, err)
		}

		result = append(result, &model.RecipeWithIngredients{
			Recipe:      recipe,
			Ingredients: ingredients,
		})
	}

	return result, total, nil
}

// AddRecipeToUserBook adds a recipe to user's recipe book
func (r *recipeRepository) AddRecipeToUserBook(ctx context.Context, userID uuid.UUID, recipeID int) error {
	query := `
		INSERT INTO recipe.user_recipes (user_id, recipe_id, added_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, recipe_id) DO NOTHING
	`

	_, err := r.db.ExecContext(ctx, query, userID, recipeID, time.Now())
	if err != nil {
		return fmt.Errorf("failed to add recipe to user book: %w", err)
	}

	return nil
}

// GetUserRecipeBook retrieves recipes from user's recipe book
func (r *recipeRepository) GetUserRecipeBook(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*model.RecipeWithIngredients, int, error) {
	// Get total count
	var total int
	countQuery := `SELECT COUNT(*) FROM recipe.user_recipes WHERE user_id = $1`
	err := r.db.QueryRowContext(ctx, countQuery, userID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count user recipe book: %w", err)
	}

	// Get paginated recipes
	query := `
		SELECT 
			r.id, r.user_id, r.name, r.description, r.category, r.is_public,
			r.total_weight, r.created_at, r.updated_at
		FROM recipe.user_recipes ur
		JOIN recipe.recipes r ON ur.recipe_id = r.id
		WHERE ur.user_id = $1
		ORDER BY ur.added_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query user recipe book: %w", err)
	}
	defer rows.Close()

	var recipes []*model.Recipe
	for rows.Next() {
		var recipe model.Recipe
		err := rows.Scan(
			&recipe.ID,
			&recipe.UserID,
			&recipe.Name,
			&recipe.Description,
			&recipe.Category,
			&recipe.IsPublic,
			&recipe.TotalWeight,
			&recipe.CreatedAt,
			&recipe.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan recipe: %w", err)
		}
		recipes = append(recipes, &recipe)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating recipe rows: %w", err)
	}

	// Get ingredients for each recipe
	result := make([]*model.RecipeWithIngredients, 0, len(recipes))
	for _, recipe := range recipes {
		ingredients, err := r.getRecipeIngredients(ctx, recipe.ID)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to get ingredients for recipe %d: %w", recipe.ID, err)
		}

		result = append(result, &model.RecipeWithIngredients{
			Recipe:      recipe,
			Ingredients: ingredients,
		})
	}

	return result, total, nil
}

// Close closes the database connection
func (r *recipeRepository) Close() error {
	return r.db.Close()
}
