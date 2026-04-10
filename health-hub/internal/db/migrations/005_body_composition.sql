ALTER TABLE body_measurements ADD COLUMN IF NOT EXISTS skeletal_muscle_mass_kg DOUBLE PRECISION;
ALTER TABLE body_measurements ADD COLUMN IF NOT EXISTS body_fat_mass_kg DOUBLE PRECISION;
