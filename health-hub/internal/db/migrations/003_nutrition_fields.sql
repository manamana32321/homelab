-- Add name and notes columns to nutrition_records
ALTER TABLE nutrition_records ADD COLUMN IF NOT EXISTS name TEXT;
ALTER TABLE nutrition_records ADD COLUMN IF NOT EXISTS notes TEXT;
