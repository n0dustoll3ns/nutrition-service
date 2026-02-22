-- Initialize database with all schemas and tables
-- This file is used by Docker Compose to initialize the database

-- Create auth schema
CREATE SCHEMA IF NOT EXISTS auth;

-- Set search path to auth schema
SET search_path TO auth;

-- Users table
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    is_active BOOLEAN DEFAULT TRUE,
    is_verified BOOLEAN DEFAULT FALSE,
    last_login_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Indexes for users table
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_is_active ON users(is_active);
CREATE INDEX idx_users_created_at ON users(created_at);

-- Refresh tokens table
CREATE TABLE refresh_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) NOT NULL,
    device_info TEXT,
    ip_address INET,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    revoked_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(user_id, device_info)
);

-- Indexes for refresh tokens
CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);
CREATE INDEX idx_refresh_tokens_token_hash ON refresh_tokens(token_hash);
CREATE INDEX idx_refresh_tokens_expires_at ON refresh_tokens(expires_at);

-- Password reset tokens table
CREATE TABLE password_reset_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    used_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for password reset tokens
CREATE INDEX idx_password_reset_tokens_token_hash ON password_reset_tokens(token_hash);
CREATE INDEX idx_password_reset_tokens_expires_at ON password_reset_tokens(expires_at);

-- Revoked access tokens table (for token blacklisting)
CREATE TABLE revoked_access_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    token_id VARCHAR(100) NOT NULL,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    revoked_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    reason VARCHAR(255)
);

-- Indexes for revoked tokens
CREATE INDEX idx_revoked_tokens_token_id ON revoked_access_tokens(token_id);
CREATE INDEX idx_revoked_tokens_expires_at ON revoked_access_tokens(expires_at);

-- Audit log table
CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    action VARCHAR(100) NOT NULL,
    resource_type VARCHAR(100),
    resource_id UUID,
    ip_address INET,
    user_agent TEXT,
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for audit logs
CREATE INDEX idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_action ON audit_logs(action);
CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at);

-- Create updated_at trigger function
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create trigger for users table
CREATE TRIGGER update_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Reset search path
RESET search_path;

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