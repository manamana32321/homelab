-- Health Hub schema initialization
-- Requires TimescaleDB extension

CREATE EXTENSION IF NOT EXISTS timescaledb;

-- Time-series metrics (steps, heart_rate, spo2, calories, distance, hydration)
CREATE TABLE IF NOT EXISTS health_metrics (
    time        TIMESTAMPTZ      NOT NULL,
    metric_type TEXT             NOT NULL,
    value       DOUBLE PRECISION NOT NULL,
    unit        TEXT             NOT NULL,
    source      TEXT             DEFAULT 'samsung_health',
    metadata    JSONB
);
SELECT create_hypertable('health_metrics', 'time', if_not_exists => TRUE);
CREATE INDEX IF NOT EXISTS idx_metrics_type_time ON health_metrics (metric_type, time DESC);

-- Sleep sessions
CREATE TABLE IF NOT EXISTS sleep_sessions (
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    start_time TIMESTAMPTZ NOT NULL,
    end_time   TIMESTAMPTZ NOT NULL,
    duration_m INT         NOT NULL,
    stages     JSONB       NOT NULL DEFAULT '[]'
);
CREATE INDEX IF NOT EXISTS idx_sleep_start ON sleep_sessions (start_time DESC);

-- Exercise sessions
CREATE TABLE IF NOT EXISTS exercise_sessions (
    id            BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    exercise_type TEXT        NOT NULL,
    start_time    TIMESTAMPTZ NOT NULL,
    end_time      TIMESTAMPTZ NOT NULL,
    duration_m    INT         NOT NULL,
    calories_kcal DOUBLE PRECISION,
    distance_m    DOUBLE PRECISION,
    metadata      JSONB
);
CREATE INDEX IF NOT EXISTS idx_exercise_start ON exercise_sessions (start_time DESC);

-- Nutrition records
CREATE TABLE IF NOT EXISTS nutrition_records (
    id        BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    time      TIMESTAMPTZ NOT NULL,
    meal_type TEXT,
    calories  DOUBLE PRECISION,
    protein_g DOUBLE PRECISION,
    fat_g     DOUBLE PRECISION,
    carbs_g   DOUBLE PRECISION,
    metadata  JSONB
);
CREATE INDEX IF NOT EXISTS idx_nutrition_time ON nutrition_records (time DESC);

-- Body measurements (hypertable for time-series queries)
CREATE TABLE IF NOT EXISTS body_measurements (
    time         TIMESTAMPTZ      NOT NULL,
    weight_kg    DOUBLE PRECISION,
    body_fat_pct DOUBLE PRECISION,
    lean_mass_kg DOUBLE PRECISION
);
SELECT create_hypertable('body_measurements', 'time', if_not_exists => TRUE);
