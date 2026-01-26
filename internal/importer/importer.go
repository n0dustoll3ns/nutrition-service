package importer

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
)

// Config holds importer configuration
type Config struct {
	DatabaseURL string
	JSONPath    string
	Schema      string
}

// DefaultConfig returns default configuration
func DefaultConfig() Config {
	return Config{
		DatabaseURL: "postgres://postgres:postgres@localhost:5432/nutrition_db?sslmode=disable",
		JSONPath:    "/app/data/foods.json",
		Schema:      "nutrition",
	}
}

// Importer handles food data import
type Importer struct {
	config Config
	db     *sql.DB
	logger *log.Logger
}

// New creates a new importer instance
func New(config Config) *Importer {
	return &Importer{
		config: config,
		logger: log.New(os.Stdout, "[importer] ", log.LstdFlags),
	}
}

// Run executes the import process
func (i *Importer) Run() error {
	startTime := time.Now()
	i.logger.Printf("Starting USDA food import at %s", startTime.Format(time.RFC3339))
	defer func() {
		i.logger.Printf("Import completed in %v", time.Since(startTime))
	}()

	// Connect to database
	if err := i.connect(); err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer i.db.Close()

	// Create tables (will drop existing data due to CASCADE)
	if err := i.createTables(); err != nil {
		i.logger.Printf("Warning: failed to create tables: %v", err)
		return fmt.Errorf("failed to create tables: %w", err)
	}

	// Import data
	if err := i.importData(); err != nil {
		i.logger.Printf("Warning: failed to import data: %v", err)
		return fmt.Errorf("failed to import data: %w", err)
	}

	return nil
}

// Connect to the database
func (i *Importer) connect() error {
	i.logger.Printf("Connecting to database: %s", i.config.DatabaseURL)
	db, err := sql.Open("postgres", i.config.DatabaseURL)
	if err != nil {
		return fmt.Errorf("failed to open database connection: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	i.db = db
	i.logger.Println("Database connection established")
	return nil
}

// Create tables in the nutrition schema
func (i *Importer) createTables() error {
	i.logger.Println("Creating tables in nutrition schema...")

	// Set search path to nutrition schema
	if _, err := i.db.Exec(fmt.Sprintf("SET search_path TO %s", i.config.Schema)); err != nil {
		return fmt.Errorf("failed to set search path: %w", err)
	}

	queries := []string{
		// Drop existing tables (CASCADE will handle dependencies)
		`DROP TABLE IF EXISTS food_portions CASCADE`,
		`DROP TABLE IF EXISTS food_attributes CASCADE`,
		`DROP TABLE IF EXISTS food_nutrients CASCADE`,
		`DROP TABLE IF EXISTS input_foods CASCADE`,
		`DROP TABLE IF EXISTS foods CASCADE`,

		// Create tables
		`CREATE TABLE foods (
			fdc_id INTEGER PRIMARY KEY,
			description TEXT NOT NULL,
			data_type TEXT,
			food_class TEXT,
			publication_date TEXT
		)`,

		`CREATE TABLE input_foods (
			id SERIAL PRIMARY KEY,
			fdc_id INTEGER REFERENCES foods(fdc_id) ON DELETE CASCADE,
			src_name TEXT,
			src_id INTEGER,
			src_table TEXT,
			src_date TEXT
		)`,

		`CREATE TABLE food_portions (
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
		)`,

		`CREATE TABLE food_attributes (
			id SERIAL PRIMARY KEY,
			fdc_id INTEGER REFERENCES foods(fdc_id) ON DELETE CASCADE,
			seq_num INTEGER,
			name TEXT,
			value TEXT,
			unit TEXT,
			data_type TEXT,
			derivation_id TEXT
		)`,

		`CREATE TABLE food_nutrients (
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
		)`,

		// Create indexes
		`CREATE INDEX idx_food_nutrients_fdc ON food_nutrients(fdc_id)`,
		`CREATE INDEX idx_food_nutrients_nutrient ON food_nutrients(nutrient_id)`,
		`CREATE INDEX idx_foods_description ON foods(description)`,
		`CREATE INDEX idx_food_portions_fdc ON food_portions(fdc_id)`,
		`CREATE INDEX idx_food_attributes_fdc ON food_attributes(fdc_id)`,
	}

	for _, query := range queries {
		if _, err := i.db.Exec(query); err != nil {
			return fmt.Errorf("failed to execute query %s: %w", query, err)
		}
	}

	i.logger.Println("Tables created successfully")
	return nil
}

// Import data from JSON file
func (i *Importer) importData() error {
	i.logger.Printf("Reading JSON file: %s", i.config.JSONPath)

	data, err := os.ReadFile(i.config.JSONPath)
	if err != nil {
		return fmt.Errorf("failed to read JSON file: %w", err)
	}

	var root struct {
		FoundationFoods []FoundationFood `json:"FoundationFoods"`
	}

	if err := json.Unmarshal(data, &root); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}

	i.logger.Printf("Found %d foods to import", len(root.FoundationFoods))

	// Set search path
	if _, err := i.db.Exec(fmt.Sprintf("SET search_path TO %s", i.config.Schema)); err != nil {
		return fmt.Errorf("failed to set search path: %w", err)
	}

	tx, err := i.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	totalFoods, totalNutrients, totalPortions, totalAttrs := 0, 0, 0, 0

	for idx, food := range root.FoundationFoods {
		// Insert food
		_, err := tx.Exec(`
			INSERT INTO foods (fdc_id, description, data_type, food_class, publication_date)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (fdc_id) DO NOTHING`,
			food.FdcId, food.Description, food.DataType, food.FoodClass, food.PublicationDate)
		if err != nil {
			i.logger.Printf("Error inserting food %d: %v", food.FdcId, err)
			continue
		}

		// Insert input foods
		for _, input := range food.InputFoods {
			_, _ = tx.Exec(`
				INSERT INTO input_foods (fdc_id, src_name, src_id, src_table, src_date)
				VALUES ($1, $2, $3, $4, $5)`,
				food.FdcId, input.SrcName, input.SrcId, input.SrcTable, input.SrcDate)
		}

		// Insert food portions
		for _, portion := range food.FoodPortions {
			_, err := tx.Exec(`
				INSERT INTO food_portions (id, fdc_id, seq_num, amount, unit_name, grams, 
					data_points, derivation_id, portion_name, portion_desc)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
				ON CONFLICT (id) DO NOTHING`,
				portion.Id, food.FdcId, portion.SeqNum, portion.Amount, portion.UnitName,
				portion.Grams, portion.DataPoints, portion.DerivationId, portion.PortionName, portion.PortionDesc)
			if err != nil {
				i.logger.Printf("Error inserting portion %d for food %d: %v", portion.Id, food.FdcId, err)
			} else {
				totalPortions++
			}
		}

		// Insert food attributes
		for _, attr := range food.FoodAttributes {
			_, _ = tx.Exec(`
				INSERT INTO food_attributes (fdc_id, seq_num, name, value, unit, data_type, derivation_id)
				VALUES ($1, $2, $3, $4, $5, $6, $7)`,
				food.FdcId, attr.SeqNum, attr.Name, attr.Value, attr.Unit, attr.DataType, attr.DerivationId)
			totalAttrs++
		}

		// Insert food nutrients
		for _, nutrient := range food.FoodNutrients {
			derivationCode, derivationDesc := "", ""
			if nutrient.FoodNutrientDerivation != nil {
				derivationCode = nutrient.FoodNutrientDerivation.Code
				derivationDesc = nutrient.FoodNutrientDerivation.Description
			}

			_, err := tx.Exec(`
				INSERT INTO food_nutrients (
					id, fdc_id, nutrient_id, nutrient_name, nutrient_number, unit_name,
					amount, data_points, min_val, max_val, median, derivation_code, derivation_desc
				) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
				ON CONFLICT (id) DO NOTHING`,
				nutrient.Id, food.FdcId, nutrient.Nutrient.Id, nutrient.Nutrient.Name,
				nutrient.Nutrient.Number, nutrient.Nutrient.UnitName, nutrient.Amount,
				nutrient.DataPoints, nutrient.Min, nutrient.Max, nutrient.Median,
				derivationCode, derivationDesc)
			if err != nil {
				// Ignore duplicate nutrients
			} else {
				totalNutrients++
			}
		}

		totalFoods++
		if idx%100 == 0 {
			i.logger.Printf("Processed %d/%d foods", idx+1, len(root.FoundationFoods))
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	i.logger.Printf("Import completed successfully:")
	i.logger.Printf("  Foods: %d", totalFoods)
	i.logger.Printf("  Portions: %d", totalPortions)
	i.logger.Printf("  Attributes: %d", totalAttrs)
	i.logger.Printf("  Nutrients: %d", totalNutrients)

	return nil
}