-- Migration to update existing food entries with calculated nutrients
-- This migration calculates and fills in missing nutrient values for existing food entries

-- Create a temporary function to calculate nutrients for a food entry
CREATE OR REPLACE FUNCTION nutrition.calculate_food_entry_nutrients(
    p_fdc_id INTEGER,
    p_amount_grams DECIMAL(10,2)
) RETURNS TABLE (
    calories DECIMAL(10,2),
    protein DECIMAL(10,2),
    fat DECIMAL(10,2),
    carbs DECIMAL(10,2)
) AS $$
BEGIN
    RETURN QUERY
    WITH nutrient_values AS (
        SELECT 
            nutrient_id,
            amount
        FROM nutrition.food_nutrients
        WHERE fdc_id = p_fdc_id 
            AND nutrient_id IN (1008, 1003, 1004, 1005) -- Energy, Protein, Fat, Carbs
    )
    SELECT 
        COALESCE((SELECT amount FROM nutrient_values WHERE nutrient_id = 1008) / 100.0 * p_amount_grams, 0) as calories,
        COALESCE((SELECT amount FROM nutrient_values WHERE nutrient_id = 1003) / 100.0 * p_amount_grams, 0) as protein,
        COALESCE((SELECT amount FROM nutrient_values WHERE nutrient_id = 1004) / 100.0 * p_amount_grams, 0) as fat,
        COALESCE((SELECT amount FROM nutrient_values WHERE nutrient_values.nutrient_id = 1005) / 100.0 * p_amount_grams, 0) as carbs;
END;
$$ LANGUAGE plpgsql;

-- Update existing food entries where calculated values are NULL and fdc_id is not NULL
UPDATE diary.food_entries fe
SET 
    calculated_calories = CASE 
        WHEN fe.calculated_calories IS NULL THEN calc.calories
        ELSE fe.calculated_calories
    END,
    calculated_protein = CASE 
        WHEN fe.calculated_protein IS NULL THEN calc.protein
        ELSE fe.calculated_protein
    END,
    calculated_fat = CASE 
        WHEN fe.calculated_fat IS NULL THEN calc.fat
        ELSE fe.calculated_fat
    END,
    calculated_carbs = CASE 
        WHEN fe.calculated_carbs IS NULL THEN calc.carbs
        ELSE fe.calculated_carbs
    END
FROM (
    SELECT 
        fe.id,
        calc.calories,
        calc.protein,
        calc.fat,
        calc.carbs
    FROM diary.food_entries fe
    CROSS JOIN LATERAL nutrition.calculate_food_entry_nutrients(fe.fdc_id, fe.amount_grams) calc
    WHERE fe.fdc_id IS NOT NULL
        AND (
            fe.calculated_calories IS NULL 
            OR fe.calculated_protein IS NULL 
            OR fe.calculated_fat IS NULL 
            OR fe.calculated_carbs IS NULL
        )
) calc
WHERE fe.id = calc.id;

-- Drop the temporary function
DROP FUNCTION nutrition.calculate_food_entry_nutrients(INTEGER, DECIMAL);

-- Log the number of updated records
DO $$
DECLARE
    updated_count INTEGER;
BEGIN
    GET DIAGNOSTICS updated_count = ROW_COUNT;
    RAISE NOTICE 'Updated % food entries with calculated nutrients', updated_count;
END $$;