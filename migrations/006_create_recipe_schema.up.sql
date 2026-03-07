-- Create recipe schema
CREATE SCHEMA IF NOT EXISTS recipe;

-- Set search path to recipe schema
SET search_path TO recipe;

-- Recipes table
CREATE TABLE recipes (
    id SERIAL PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    category VARCHAR(100),
    is_public BOOLEAN NOT NULL DEFAULT false,
    total_weight DECIMAL(10,2) NOT NULL DEFAULT 0 CHECK (total_weight >= 0),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Recipe ingredients table
CREATE TABLE recipe_ingredients (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    recipe_id INTEGER NOT NULL REFERENCES recipe.recipes(id) ON DELETE CASCADE,
    fdc_id INTEGER REFERENCES nutrition.foods(fdc_id) ON DELETE SET NULL,
    custom_food_name VARCHAR(255),
    amount_grams DECIMAL(10,2) NOT NULL CHECK (amount_grams > 0),
    -- Cached nutrients per 100g of this ingredient
    calories_per_100g DECIMAL(10,2),
    protein_per_100g DECIMAL(10,2),
    fat_per_100g DECIMAL(10,2),
    carbs_per_100g DECIMAL(10,2),
    
    -- One of fdc_id or custom_food_name must be set
    CONSTRAINT chk_ingredient_source CHECK (
        (fdc_id IS NOT NULL AND custom_food_name IS NULL) OR 
        (fdc_id IS NULL AND custom_food_name IS NOT NULL)
    )
);

-- User recipes table (recipes added to user's recipe book)
CREATE TABLE user_recipes (
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    recipe_id INTEGER NOT NULL REFERENCES recipe.recipes(id) ON DELETE CASCADE,
    added_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (user_id, recipe_id)
);

-- Indexes for performance
CREATE INDEX idx_recipes_user_id ON recipes(user_id);
CREATE INDEX idx_recipes_is_public ON recipes(is_public) WHERE is_public = true;
CREATE INDEX idx_recipes_category ON recipes(category) WHERE category IS NOT NULL;
CREATE INDEX idx_recipes_name ON recipes(name);

CREATE INDEX idx_recipe_ingredients_recipe_id ON recipe_ingredients(recipe_id);
CREATE INDEX idx_recipe_ingredients_fdc_id ON recipe_ingredients(fdc_id) WHERE fdc_id IS NOT NULL;

CREATE INDEX idx_user_recipes_user_id ON user_recipes(user_id);
CREATE INDEX idx_user_recipes_recipe_id ON user_recipes(recipe_id);

-- Function to update recipe total weight
CREATE OR REPLACE FUNCTION update_recipe_total_weight()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' OR TG_OP = 'UPDATE' THEN
        UPDATE recipe.recipes 
        SET total_weight = (
            SELECT COALESCE(SUM(amount_grams), 0)
            FROM recipe.recipe_ingredients
            WHERE recipe_id = NEW.recipe_id
        ),
        updated_at = NOW()
        WHERE id = NEW.recipe_id;
    ELSIF TG_OP = 'DELETE' THEN
        UPDATE recipe.recipes 
        SET total_weight = (
            SELECT COALESCE(SUM(amount_grams), 0)
            FROM recipe.recipe_ingredients
            WHERE recipe_id = OLD.recipe_id
        ),
        updated_at = NOW()
        WHERE id = OLD.recipe_id;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

-- Trigger to update recipe total weight when ingredients change
CREATE TRIGGER trigger_update_recipe_total_weight
AFTER INSERT OR UPDATE OR DELETE ON recipe.recipe_ingredients
FOR EACH ROW EXECUTE FUNCTION update_recipe_total_weight();

-- Function to update recipe updated_at timestamp
CREATE OR REPLACE FUNCTION update_recipe_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to update recipe updated_at timestamp
CREATE TRIGGER trigger_update_recipe_timestamp
BEFORE UPDATE ON recipe.recipes
FOR EACH ROW EXECUTE FUNCTION update_recipe_timestamp();

-- Modify diary.food_entries to support recipe_id
ALTER TABLE diary.food_entries 
ADD COLUMN recipe_id INTEGER REFERENCES recipe.recipes(id) ON DELETE SET NULL;

-- Update constraint to support three possible sources
ALTER TABLE diary.food_entries 
DROP CONSTRAINT IF EXISTS chk_food_source;

ALTER TABLE diary.food_entries 
ADD CONSTRAINT chk_food_source CHECK (
    (fdc_id IS NOT NULL AND custom_food_name IS NULL AND recipe_id IS NULL) OR 
    (fdc_id IS NULL AND custom_food_name IS NOT NULL AND recipe_id IS NULL) OR
    (fdc_id IS NULL AND custom_food_name IS NULL AND recipe_id IS NOT NULL)
);

-- Add index for recipe_id in food_entries
CREATE INDEX idx_food_entries_recipe ON diary.food_entries(recipe_id) WHERE recipe_id IS NOT NULL;

-- Reset search path
RESET search_path;