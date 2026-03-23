-- Unique constraint: same (time, metric_type) = one row only.
-- HC Webhook sends overlapping data each sync — upsert keeps latest value.
CREATE UNIQUE INDEX IF NOT EXISTS idx_metrics_dedup
    ON health_metrics (time, metric_type);

CREATE UNIQUE INDEX IF NOT EXISTS idx_body_dedup
    ON body_measurements (time);
