-- Create nutrition schema
CREATE SCHEMA IF NOT EXISTS nutrition;

-- Set search path to nutrition schema
SET search_path TO nutrition;

-- Foods table (main products)
CREATE TABLE foods (
    fdc_id INTEGER PRIMARY KEY,
    description TEXT NOT NULL,
    data_type TEXT,
    food_class TEXT,
    publication_date TEXT
);

-- Input foods table
CREATE TABLE input_foods (
    id SERIAL PRIMARY KEY,
    fdc_id INTEGER REFERENCES foods(fdc_id) ON DELETE CASCADE,
    src_name TEXT,
    src_id INTEGER,
    src_table TEXT,
    src_date TEXT
);

-- Food portions table
CREATE TABLE food_portions (
    id INTEGER PRIMARY KEY,
    fdc_id INTEGER REFERENCES foods(fdc_id) ON DELETE CASCADE,
    seq_num INTEGER,
    amount DOUBLE PRECISION,
    unit_name TEXT,
    grams DOUBLE PRECISION,
    data_points INTEGER,
    derivation_id TEXT,
    portion_name TEXT,
    portion_desc TEXT
);

-- Food attributes table
CREATE TABLE food_attributes (
    id SERIAL PRIMARY KEY,
    fdc_id INTEGER REFERENCES foods(fdc_id) ON DELETE CASCADE,
    seq_num INTEGER,
    name TEXT,
    value TEXT,
    unit TEXT,
    data_type TEXT,
    derivation_id TEXT
);

-- Food nutrients table
CREATE TABLE food_nutrients (
    id INTEGER PRIMARY KEY,
    fdc_id INTEGER REFERENCES foods(fdc_id) ON DELETE CASCADE,
    nutrient_id INTEGER NOT NULL,
    nutrient_name TEXT,
    nutrient_number TEXT,
    unit_name TEXT,
    amount DOUBLE PRECISION,
    data_points INTEGER,
    min_val DOUBLE PRECISION,
    max_val DOUBLE PRECISION,
    median DOUBLE PRECISION,
    derivation_code TEXT,
    derivation_desc TEXT
);

-- Indexes for performance
CREATE INDEX idx_food_nutrients_fdc ON food_nutrients(fdc_id);
CREATE INDEX idx_food_nutrients_nutrient ON food_nutrients(nutrient_id);
CREATE INDEX idx_foods_description ON foods(description);
CREATE INDEX idx_food_portions_fdc ON food_portions(fdc_id);
CREATE INDEX idx_food_attributes_fdc ON food_attributes(fdc_id);

-- Reset search path
RESET search_path;