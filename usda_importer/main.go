package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
)

// Полная структура Foundation Foods JSON [file:146]
type FoundationFood struct {
	FdcId            int             `json:"fdcId"`
	Description      string          `json:"description"`
	DataType         string          `json:"dataType"`
	FoodClass        string          `json:"foodClass"`
	PublicationDate  string          `json:"publicationDate"`
	FoodNutrients    []FoodNutrient  `json:"foodNutrients"`
	AllNutrientNames []string        `json:"allNutrientNames,omitempty"`
	InputFoods       []InputFood     `json:"inputFoods,omitempty"`
	FoodPortions     []FoodPortion   `json:"foodPortions"`
	FoodAttributes   []FoodAttribute `json:"foodAttributes"`
}

type FoodNutrient struct {
	Type                   string      `json:"type"`
	Id                     int         `json:"id"`
	Nutrient               Nutrient    `json:"nutrient"`
	Amount                 float64     `json:"amount,omitempty"`
	DataPoints             int         `json:"dataPoints,omitempty"`
	Min                    float64     `json:"min,omitempty"`
	Max                    float64     `json:"max,omitempty"`
	Median                 float64     `json:"median,omitempty"`
	FoodNutrientDerivation *Derivation `json:"foodNutrientDerivation,omitempty"`
}

type Nutrient struct {
	Id       int    `json:"id"`
	Number   string `json:"number"`
	Name     string `json:"name"`
	Rank     int    `json:"rank"`
	UnitName string `json:"unitName"`
}

type Derivation struct {
	Code               string  `json:"code"`
	Description        string  `json:"description"`
	FoodNutrientSource *Source `json:"foodNutrientSource"`
}

type Source struct {
	Id          int    `json:"id"`
	Code        string `json:"code"`
	Description string `json:"description"`
}

type FoodPortion struct {
	Id           int     `json:"id"`
	SeqNum       int     `json:"seqNum"`
	Amount       float64 `json:"amount"`
	UnitName     string  `json:"unitName"`
	Grams        float64 `json:"gramWeight"`
	DataPoints   int     `json:"dataPoints"`
	DerivationId string  `json:"derivationId"`
	PortionName  string  `json:"portionName"`
	PortionDesc  string  `json:"portionDescription"`
}

type FoodAttribute struct {
	SeqNum       int    `json:"seqNum"`
	Name         string `json:"name"`
	Value        string `json:"value"`
	Unit         string `json:"unit"`
	DataType     string `json:"dataType"`
	DerivationId string `json:"derivationId"`
}

type InputFood struct {
	SrcName  string `json:"srcName"`
	SrcId    int    `json:"srcId"`
	SrcTable string `json:"srcTable"`
	SrcDate  string `json:"srcDate"`
}

func main() {
	connStr := "host=localhost user=fominyhdenis dbname=usda_db sslmode=disable"

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Ошибка подключения:", err)
	}
	defer db.Close()

	if err := createTables(db); err != nil {
		log.Fatal("Ошибка создания таблиц:", err)
	}

	if err := importFoundationFoods(db, "FoodData_Central_foundation_food_json_2025-12-18.json"); err != nil {
		log.Fatal("Ошибка импорта:", err)
	}

	fmt.Println("✅ Полный импорт Foundation Foods завершён!")
}

func createTables(db *sql.DB) error {
	queries := []string{
		// Очищаем старые таблицы
		`DROP TABLE IF EXISTS food_portions CASCADE`,
		`DROP TABLE IF EXISTS food_attributes CASCADE`,
		`DROP TABLE IF EXISTS food_nutrients CASCADE`,
		`DROP TABLE IF EXISTS input_foods CASCADE`,
		`DROP TABLE IF EXISTS foods CASCADE`,

		// Основная таблица продуктов
		`CREATE TABLE foods (
            fdc_id INTEGER PRIMARY KEY,
            description TEXT NOT NULL,
            data_type TEXT,
            food_class TEXT,
            publication_date TEXT
        )`,

		// Связующие таблицы
		`CREATE TABLE input_foods (
            id SERIAL PRIMARY KEY,
            fdc_id INTEGER REFERENCES foods(fdc_id),
            src_name TEXT,
            src_id INTEGER,
            src_table TEXT,
            src_date TEXT
        )`,

		`CREATE TABLE food_portions (
            id INTEGER PRIMARY KEY,
            fdc_id INTEGER REFERENCES foods(fdc_id),
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
            fdc_id INTEGER REFERENCES foods(fdc_id),
            seq_num INTEGER,
            name TEXT,
            value TEXT,
            unit TEXT,
            data_type TEXT,
            derivation_id TEXT
        )`,

		`CREATE TABLE food_nutrients (
            id INTEGER PRIMARY KEY,
            fdc_id INTEGER REFERENCES foods(fdc_id),
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

		// Индексы для быстрого поиска
		`CREATE INDEX idx_food_nutrients_fdc ON food_nutrients(fdc_id)`,
		`CREATE INDEX idx_food_nutrients_nutrient ON food_nutrients(nutrient_id)`,
		`CREATE INDEX idx_foods_description ON foods(description)`,
		`CREATE INDEX idx_food_portions_fdc ON food_portions(fdc_id)`,
		`CREATE INDEX idx_food_attributes_fdc ON food_attributes(fdc_id)`,
	}

	for _, query := range queries {
		if _, err := db.Exec(query); err != nil {
			return fmt.Errorf("ошибка выполнения %s: %w", query, err)
		}
	}
	fmt.Println("✅ Все таблицы созданы (с порциями и атрибутами)")
	return nil
}

func importFoundationFoods(db *sql.DB, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("не могу прочитать %s: %w", path, err)
	}

	var root struct {
		FoundationFoods []FoundationFood `json:"FoundationFoods"`
	}

	if err := json.Unmarshal(data, &root); err != nil {
		return fmt.Errorf("ошибка парсинга JSON: %w", err)
	}

	fmt.Printf("🔄 Загружаю %d продуктов с порциями и атрибутами...\n", len(root.FoundationFoods))

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	totalFoods, totalNutrients, totalPortions, totalAttrs := 0, 0, 0, 0

	for _, food := range root.FoundationFoods {
		// 1. Продукт
		_, err := tx.Exec(`
            INSERT INTO foods (fdc_id, description, data_type, food_class, publication_date)
            VALUES ($1, $2, $3, $4, $5)
            ON CONFLICT (fdc_id) DO NOTHING`,
			food.FdcId, food.Description, food.DataType, food.FoodClass, food.PublicationDate)
		if err != nil {
			log.Printf("Ошибка продукта %d: %v", food.FdcId, err)
			continue
		}

		// 2. InputFoods (если есть)
		for _, input := range food.InputFoods {
			_, _ = tx.Exec(`
                INSERT INTO input_foods (fdc_id, src_name, src_id, src_table, src_date)
                VALUES ($1, $2, $3, $4, $5)`,
				food.FdcId, input.SrcName, input.SrcId, input.SrcTable, input.SrcDate)
		}

		// 3. Порции (FoodPortions)
		for _, portion := range food.FoodPortions {
			_, err := tx.Exec(`
                INSERT INTO food_portions (id, fdc_id, seq_num, amount, unit_name, grams, 
                    data_points, derivation_id, portion_name, portion_desc)
                VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
                ON CONFLICT (id) DO NOTHING`,
				portion.Id, food.FdcId, portion.SeqNum, portion.Amount, portion.UnitName,
				portion.Grams, portion.DataPoints, portion.DerivationId, portion.PortionName, portion.PortionDesc)
			if err != nil {
				log.Printf("Ошибка порции %d для %d: %v", portion.Id, food.FdcId, err)
			} else {
				totalPortions++
			}
		}

		// 4. Атрибуты (FoodAttributes)
		for _, attr := range food.FoodAttributes {
			_, _ = tx.Exec(`
                INSERT INTO food_attributes (fdc_id, seq_num, name, value, unit, data_type, derivation_id)
                VALUES ($1, $2, $3, $4, $5, $6, $7)`,
				food.FdcId, attr.SeqNum, attr.Name, attr.Value, attr.Unit, attr.DataType, attr.DerivationId)
			totalAttrs++
		}

		// 5. Нутриенты (самое важное)
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
				// Игнорируем дубликаты нутриентов
			} else {
				totalNutrients++
			}
		}

		totalFoods++
		if totalFoods%50 == 0 {
			fmt.Printf("Обработано %d/%d продуктов\r", totalFoods, len(root.FoundationFoods))
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	fmt.Printf("\n✅ Импорт завершён!\n")
	fmt.Printf("  📦 Продуктов: %d\n", totalFoods)
	fmt.Printf("  🥄 Порций: %d\n", totalPortions)
	fmt.Printf("  🏷️ Атрибутов: %d\n", totalAttrs)
	fmt.Printf("  🧪 Нутриентов: %d\n", totalNutrients)

	return nil
}
