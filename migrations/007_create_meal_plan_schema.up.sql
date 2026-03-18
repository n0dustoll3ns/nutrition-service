-- Create meal_plan schema
CREATE SCHEMA IF NOT EXISTS meal_plan;

-- Set search path to meal_plan schema
SET search_path TO meal_plan;

-- Meal plans table
CREATE TABLE meal_plans (
    id SERIAL PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    days_count INTEGER NOT NULL CHECK (days_count > 0),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Meal plan entries table
CREATE TABLE meal_plan_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    meal_plan_id INTEGER NOT NULL REFERENCES meal_plan.meal_plans(id) ON DELETE CASCADE,
    day_number INTEGER NOT NULL CHECK (day_number > 0),
    meal_type VARCHAR(20) NOT NULL CHECK (meal_type IN ('breakfast', 'brunch', 'lunch', 'afternoon_snack', 'dinner', 'snack')),
    fdc_id INTEGER REFERENCES nutrition.foods(fdc_id) ON DELETE SET NULL,
    custom_food_name VARCHAR(255),
    recipe_id INTEGER REFERENCES recipe.recipes(id) ON DELETE SET NULL,
    amount_grams DECIMAL(10,2) NOT NULL CHECK (amount_grams > 0),
    
    -- One of fdc_id, custom_food_name, or recipe_id must be set
    CONSTRAINT chk_meal_plan_food_source CHECK (
        (fdc_id IS NOT NULL AND custom_food_name IS NULL AND recipe_id IS NULL) OR 
        (fdc_id IS NULL AND custom_food_name IS NOT NULL AND recipe_id IS NULL) OR
        (fdc_id IS NULL AND custom_food_name IS NULL AND recipe_id IS NOT NULL)
    )
);

-- Indexes for performance
CREATE INDEX idx_meal_plans_user_id ON meal_plans(user_id);
CREATE INDEX idx_meal_plans_name ON meal_plans(name);

CREATE INDEX idx_meal_plan_entries_meal_plan_id ON meal_plan_entries(meal_plan_id);
CREATE INDEX idx_meal_plan_entries_day_number ON meal_plan_entries(day_number);
CREATE INDEX idx_meal_plan_entries_meal_type ON meal_plan_entries(meal_type);
CREATE INDEX idx_meal_plan_entries_fdc_id ON meal_plan_entries(fdc_id) WHERE fdc_id IS NOT NULL;
CREATE INDEX idx_meal_plan_entries_recipe_id ON meal_plan_entries(recipe_id) WHERE recipe_id IS NOT NULL;

-- Function to update meal plan updated_at timestamp
CREATE OR REPLACE FUNCTION update_meal_plan_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to update meal plan updated_at timestamp
CREATE TRIGGER trigger_update_meal_plan_timestamp
BEFORE UPDATE ON meal_plan.meal_plans
FOR EACH ROW EXECUTE FUNCTION update_meal_plan_timestamp();

-- Reset search path
RESET search_path;