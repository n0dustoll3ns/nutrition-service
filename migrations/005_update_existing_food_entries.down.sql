-- Down migration: This migration cannot be fully reversed as we're updating data
-- However, we can note that this is a data migration that updates calculated values
-- based on existing data, so rolling back would require restoring from backup

-- Note: This is a data migration. To roll back, you would need to restore
-- the calculated_calories, calculated_protein, calculated_fat, calculated_carbs
-- columns to their previous values from a backup.

-- Since we cannot know the previous values, we'll just log a warning
DO $$
BEGIN
    RAISE WARNING 'This is a data migration. To roll back, restore the calculated nutrient values from backup.';
END $$;