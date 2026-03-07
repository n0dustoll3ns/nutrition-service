-- Set search path to recipe schema
SET search_path TO recipe;

-- Drop triggers first
DROP TRIGGER IF EXISTS trigger_update_recipe_timestamp ON recipes;
DROP TRIGGER IF EXISTS trigger_update_recipe_total_weight ON recipe_ingredients;

-- Drop functions
DROP FUNCTION IF EXISTS update_recipe_timestamp();
DROP FUNCTION IF EXISTS update_recipe_total_weight();

-- Remove recipe_id from diary.food_entries
ALTER TABLE diary.food_entries 
DROP CONSTRAINT IF EXISTS chk_food_source;

-- Restore original constraint
ALTER TABLE diary.food_entries 
ADD CONSTRAINT chk_food_source CHECK (
    (fdc_id IS NOT NULL AND custom_food_name IS NULL) OR 
    (fdc_id IS NULL AND custom_food_name IS NOT NULL)
);

-- Drop index for recipe_id in food_entries
DROP INDEX IF EXISTS diary.idx_food_entries_recipe;

-- Remove recipe_id column from diary.food_entries
ALTER TABLE diary.food_entries 
DROP COLUMN IF EXISTS recipe_id;

-- Drop tables in reverse order of dependencies
DROP TABLE IF EXISTS user_recipes;
DROP TABLE IF EXISTS recipe_ingredients;
DROP TABLE IF EXISTS recipes;

-- Reset search path
RESET search_path;

-- Drop recipe schema
DROP SCHEMA IF EXISTS recipe;