-- Create diary schema
CREATE SCHEMA IF NOT EXISTS diary;

-- Set search path to diary schema
SET search_path TO diary;

-- Food entries table
CREATE TABLE food_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    date DATE NOT NULL,
    meal_type VARCHAR(20) NOT NULL CHECK (meal_type IN ('breakfast', 'brunch', 'lunch', 'afternoon_snack', 'dinner', 'snack')),
    fdc_id INTEGER REFERENCES nutrition.foods(fdc_id) ON DELETE SET NULL,
    custom_food_name VARCHAR(255),
    amount_grams DECIMAL(10,2) NOT NULL CHECK (amount_grams > 0),
    calculated_calories DECIMAL(10,2),
    calculated_protein DECIMAL(10,2),
    calculated_fat DECIMAL(10,2),
    calculated_carbs DECIMAL(10,2),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- One of fdc_id or custom_food_name must be set
    CONSTRAINT chk_food_source CHECK (
        (fdc_id IS NOT NULL AND custom_food_name IS NULL) OR 
        (fdc_id IS NULL AND custom_food_name IS NOT NULL)
    )
);

-- Indexes for performance
CREATE INDEX idx_food_entries_user_date ON food_entries(user_id, date);
CREATE INDEX idx_food_entries_date ON food_entries(date);
CREATE INDEX idx_food_entries_meal_type ON food_entries(meal_type);
CREATE INDEX idx_food_entries_fdc ON food_entries(fdc_id) WHERE fdc_id IS NOT NULL;
CREATE INDEX idx_food_entries_user_date_meal ON food_entries(user_id, date, meal_type);

-- Reset search path
RESET search_path;