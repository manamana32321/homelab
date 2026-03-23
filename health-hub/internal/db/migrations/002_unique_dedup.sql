-- Remove duplicate rows before adding unique constraint.
-- Keep only the first inserted row for each (time, metric_type, value) combination.
DELETE FROM health_metrics a USING health_metrics b
WHERE a.ctid > b.ctid
  AND a.time = b.time
  AND a.metric_type = b.metric_type
  AND a.value = b.value;

DELETE FROM body_measurements a USING body_measurements b
WHERE a.ctid > b.ctid
  AND a.time = b.time
  AND a.weight_kg IS NOT DISTINCT FROM b.weight_kg;

-- Now add unique constraints to prevent future duplicates.
-- TimescaleDB requires the partitioning column (time) in UNIQUE constraints.
CREATE UNIQUE INDEX IF NOT EXISTS idx_metrics_dedup
    ON health_metrics (time, metric_type, value);

CREATE UNIQUE INDEX IF NOT EXISTS idx_body_dedup
    ON body_measurements (time, weight_kg);
