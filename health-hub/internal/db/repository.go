package db

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/manamana32321/homelab/health-hub/internal/model"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Ping(ctx context.Context) error {
	return r.pool.Ping(ctx)
}

// InsertMetrics bulk-inserts time-series metrics with ON CONFLICT dedup.
func (r *Repository) InsertMetrics(ctx context.Context, metrics []model.Metric) (int, error) {
	if len(metrics) == 0 {
		return 0, nil
	}
	query := `INSERT INTO health_metrics (time, metric_type, value, unit, source, metadata)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (time, metric_type) DO UPDATE SET value = EXCLUDED.value, unit = EXCLUDED.unit, source = EXCLUDED.source, metadata = EXCLUDED.metadata`

	batch := &pgx.Batch{}
	for _, m := range metrics {
		src := m.Source
		if src == "" {
			src = "samsung_health"
		}
		var meta []byte
		if m.Metadata != nil {
			meta, _ = json.Marshal(m.Metadata)
		}
		batch.Queue(query, m.Time, m.MetricType, m.Value, m.Unit, src, meta)
	}

	br := r.pool.SendBatch(ctx, batch)
	defer br.Close()

	inserted := 0
	for range metrics {
		ct, err := br.Exec()
		if err != nil {
			return inserted, fmt.Errorf("insert metric: %w", err)
		}
		inserted += int(ct.RowsAffected())
	}
	return inserted, nil
}

// InsertSleepSessions inserts sleep sessions, deduplicating by start_time.
func (r *Repository) InsertSleepSessions(ctx context.Context, sessions []model.SleepSession) (int, error) {
	if len(sessions) == 0 {
		return 0, nil
	}

	inserted := 0
	for _, s := range sessions {
		stagesJSON, _ := json.Marshal(s.Stages)

		// Check for duplicate by start_time (within 1 minute window)
		var exists bool
		err := r.pool.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM sleep_sessions WHERE start_time BETWEEN $1::timestamptz - INTERVAL '1 minute' AND $1::timestamptz + INTERVAL '1 minute')`,
			s.StartTime).Scan(&exists)
		if err != nil {
			return inserted, fmt.Errorf("check sleep dup: %w", err)
		}
		if exists {
			continue
		}

		_, err = r.pool.Exec(ctx,
			`INSERT INTO sleep_sessions (start_time, end_time, duration_m, stages) VALUES ($1, $2, $3, $4)`,
			s.StartTime, s.EndTime, s.DurationM, stagesJSON)
		if err != nil {
			return inserted, fmt.Errorf("insert sleep: %w", err)
		}
		inserted++
	}
	return inserted, nil
}

// InsertExerciseSessions inserts exercise sessions, deduplicating by start_time + type.
func (r *Repository) InsertExerciseSessions(ctx context.Context, sessions []model.ExerciseSession) (int, error) {
	if len(sessions) == 0 {
		return 0, nil
	}

	inserted := 0
	for _, s := range sessions {
		var meta []byte
		if s.Metadata != nil {
			meta, _ = json.Marshal(s.Metadata)
		}

		var exists bool
		err := r.pool.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM exercise_sessions WHERE exercise_type = $1 AND start_time BETWEEN $2::timestamptz - INTERVAL '1 minute' AND $2::timestamptz + INTERVAL '1 minute')`,
			s.ExerciseType, s.StartTime).Scan(&exists)
		if err != nil {
			return inserted, fmt.Errorf("check exercise dup: %w", err)
		}
		if exists {
			continue
		}

		_, err = r.pool.Exec(ctx,
			`INSERT INTO exercise_sessions (exercise_type, start_time, end_time, duration_m, calories_kcal, distance_m, metadata) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			s.ExerciseType, s.StartTime, s.EndTime, s.DurationM, s.CaloriesKcal, s.DistanceM, meta)
		if err != nil {
			return inserted, fmt.Errorf("insert exercise: %w", err)
		}
		inserted++
	}
	return inserted, nil
}

// InsertNutritionRecords inserts nutrition records.
func (r *Repository) InsertNutritionRecords(ctx context.Context, records []model.NutritionRecord) (int, error) {
	if len(records) == 0 {
		return 0, nil
	}

	inserted := 0
	for _, n := range records {
		var meta []byte
		if n.Metadata != nil {
			meta, _ = json.Marshal(n.Metadata)
		}

		// Dedup by time + meal_type (within 1 minute)
		var exists bool
		err := r.pool.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM nutrition_records WHERE time BETWEEN $1::timestamptz - INTERVAL '1 minute' AND $1::timestamptz + INTERVAL '1 minute' AND COALESCE(meal_type,'') = COALESCE($2,''))`,
			n.Time, n.MealType).Scan(&exists)
		if err != nil {
			return inserted, fmt.Errorf("check nutrition dup: %w", err)
		}
		if exists {
			continue
		}

		_, err = r.pool.Exec(ctx,
			`INSERT INTO nutrition_records (time, name, meal_type, calories, protein_g, fat_g, carbs_g, notes, metadata) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
			n.Time, n.Name, n.MealType, n.Calories, n.ProteinG, n.FatG, n.CarbsG, n.Notes, meta)
		if err != nil {
			return inserted, fmt.Errorf("insert nutrition: %w", err)
		}
		inserted++
	}
	return inserted, nil
}

// InsertNutritionRecord inserts a single nutrition record and returns the ID.
func (r *Repository) InsertNutritionRecord(ctx context.Context, n model.NutritionRecord) (int64, error) {
	var meta []byte
	if n.Metadata != nil {
		meta, _ = json.Marshal(n.Metadata)
	}

	var id int64
	err := r.pool.QueryRow(ctx,
		`INSERT INTO nutrition_records (time, name, meal_type, calories, protein_g, fat_g, carbs_g, notes, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id`,
		n.Time, n.Name, n.MealType, n.Calories, n.ProteinG, n.FatG, n.CarbsG, n.Notes, meta).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("insert nutrition: %w", err)
	}
	return id, nil
}

// UpdateNutritionRecord updates an existing nutrition record by ID.
func (r *Repository) UpdateNutritionRecord(ctx context.Context, n model.NutritionRecord) error {
	var meta []byte
	if n.Metadata != nil {
		meta, _ = json.Marshal(n.Metadata)
	}

	ct, err := r.pool.Exec(ctx,
		`UPDATE nutrition_records SET time=$1, name=$2, meal_type=$3, calories=$4, protein_g=$5, fat_g=$6, carbs_g=$7, notes=$8, metadata=$9 WHERE id=$10`,
		n.Time, n.Name, n.MealType, n.Calories, n.ProteinG, n.FatG, n.CarbsG, n.Notes, meta, n.ID)
	if err != nil {
		return fmt.Errorf("update nutrition: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("record not found: %d", n.ID)
	}
	return nil
}

// DeleteNutritionRecord deletes a nutrition record by ID.
func (r *Repository) DeleteNutritionRecord(ctx context.Context, id int64) error {
	ct, err := r.pool.Exec(ctx, `DELETE FROM nutrition_records WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete nutrition: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("record not found: %d", id)
	}
	return nil
}

// GetNutritionRecord gets a single nutrition record by ID.
func (r *Repository) GetNutritionRecord(ctx context.Context, id int64) (*model.NutritionRecord, error) {
	var n model.NutritionRecord
	var metaJSON []byte
	err := r.pool.QueryRow(ctx,
		`SELECT id, time, name, meal_type, calories, protein_g, fat_g, carbs_g, notes, metadata FROM nutrition_records WHERE id = $1`, id).
		Scan(&n.ID, &n.Time, &n.Name, &n.MealType, &n.Calories, &n.ProteinG, &n.FatG, &n.CarbsG, &n.Notes, &metaJSON)
	if err != nil {
		return nil, fmt.Errorf("get nutrition: %w", err)
	}
	if metaJSON != nil {
		_ = json.Unmarshal(metaJSON, &n.Metadata)
	}
	return &n, nil
}

// QueryNutritionByType returns nutrition records filtered by meal_type.
func (r *Repository) QueryNutritionByType(ctx context.Context, q model.TimeRangeQuery, mealType string) ([]model.NutritionRecord, error) {
	query := `SELECT id, time, name, meal_type, calories, protein_g, fat_g, carbs_g, notes, metadata
		FROM nutrition_records WHERE time >= $1 AND time < $2`
	args := []interface{}{q.From, q.To}

	if mealType != "" {
		query += " AND meal_type = $3"
		args = append(args, mealType)
	}
	query += " ORDER BY time DESC"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query nutrition: %w", err)
	}
	defer rows.Close()

	var results []model.NutritionRecord
	for rows.Next() {
		var n model.NutritionRecord
		var metaJSON []byte
		if err := rows.Scan(&n.ID, &n.Time, &n.Name, &n.MealType, &n.Calories, &n.ProteinG, &n.FatG, &n.CarbsG, &n.Notes, &metaJSON); err != nil {
			return nil, fmt.Errorf("scan nutrition: %w", err)
		}
		if metaJSON != nil {
			_ = json.Unmarshal(metaJSON, &n.Metadata)
		}
		results = append(results, n)
	}
	return results, rows.Err()
}

// InsertBodyMeasurement inserts a single body measurement and returns the time.
func (r *Repository) InsertBodyMeasurement(ctx context.Context, m model.BodyMeasurement) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO body_measurements (time, weight_kg, body_fat_pct, lean_mass_kg, skeletal_muscle_mass_kg, body_fat_mass_kg)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (time) DO UPDATE SET
			weight_kg = COALESCE(EXCLUDED.weight_kg, body_measurements.weight_kg),
			body_fat_pct = COALESCE(EXCLUDED.body_fat_pct, body_measurements.body_fat_pct),
			lean_mass_kg = COALESCE(EXCLUDED.lean_mass_kg, body_measurements.lean_mass_kg),
			skeletal_muscle_mass_kg = COALESCE(EXCLUDED.skeletal_muscle_mass_kg, body_measurements.skeletal_muscle_mass_kg),
			body_fat_mass_kg = COALESCE(EXCLUDED.body_fat_mass_kg, body_measurements.body_fat_mass_kg)`,
		m.Time, m.WeightKg, m.BodyFatPct, m.LeanMassKg, m.SkeletalMuscleMassKg, m.BodyFatMassKg)
	return err
}

// InsertHealthNote inserts a health note and returns the ID.
func (r *Repository) InsertHealthNote(ctx context.Context, n model.HealthNote) (int64, error) {
	if n.Category == "" {
		n.Category = "memo"
	}
	var id int64
	err := r.pool.QueryRow(ctx,
		`INSERT INTO health_notes (time, category, text) VALUES ($1, $2, $3) RETURNING id`,
		n.Time, n.Category, n.Text).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("insert note: %w", err)
	}
	return id, nil
}

// QueryHealthNotes returns health notes filtered by time range and optional category.
func (r *Repository) QueryHealthNotes(ctx context.Context, q model.TimeRangeQuery, category string) ([]model.HealthNote, error) {
	query := `SELECT id, time, category, text FROM health_notes WHERE time >= $1 AND time < $2`
	args := []interface{}{q.From, q.To}

	if category != "" {
		query += " AND category = $3"
		args = append(args, category)
	}
	query += " ORDER BY time DESC"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query notes: %w", err)
	}
	defer rows.Close()

	var results []model.HealthNote
	for rows.Next() {
		var n model.HealthNote
		if err := rows.Scan(&n.ID, &n.Time, &n.Category, &n.Text); err != nil {
			return nil, fmt.Errorf("scan note: %w", err)
		}
		results = append(results, n)
	}
	return results, rows.Err()
}

// InsertBodyMeasurements inserts body composition data.
func (r *Repository) InsertBodyMeasurements(ctx context.Context, measurements []model.BodyMeasurement) (int, error) {
	if len(measurements) == 0 {
		return 0, nil
	}

	query := `INSERT INTO body_measurements (time, weight_kg, body_fat_pct, lean_mass_kg, skeletal_muscle_mass_kg, body_fat_mass_kg)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (time) DO UPDATE SET
			weight_kg = COALESCE(EXCLUDED.weight_kg, body_measurements.weight_kg),
			body_fat_pct = COALESCE(EXCLUDED.body_fat_pct, body_measurements.body_fat_pct),
			lean_mass_kg = COALESCE(EXCLUDED.lean_mass_kg, body_measurements.lean_mass_kg),
			skeletal_muscle_mass_kg = COALESCE(EXCLUDED.skeletal_muscle_mass_kg, body_measurements.skeletal_muscle_mass_kg),
			body_fat_mass_kg = COALESCE(EXCLUDED.body_fat_mass_kg, body_measurements.body_fat_mass_kg)`

	batch := &pgx.Batch{}
	for _, m := range measurements {
		batch.Queue(query, m.Time, m.WeightKg, m.BodyFatPct, m.LeanMassKg, m.SkeletalMuscleMassKg, m.BodyFatMassKg)
	}

	br := r.pool.SendBatch(ctx, batch)
	defer br.Close()

	inserted := 0
	for range measurements {
		ct, err := br.Exec()
		if err != nil {
			return inserted, fmt.Errorf("insert body: %w", err)
		}
		inserted += int(ct.RowsAffected())
	}
	return inserted, nil
}

// QueryMetrics returns aggregated metrics for a given type and time range.
func (r *Repository) QueryMetrics(ctx context.Context, q model.MetricsQuery) ([]model.AggregatedMetric, error) {
	interval := q.Interval
	if interval == "" {
		interval = "1h"
	}

	rows, err := r.pool.Query(ctx, `
		SELECT time_bucket($1::interval, time) AS bucket,
			AVG(value) AS avg, MIN(value) AS min, MAX(value) AS max,
			SUM(value) AS sum, COUNT(*) AS count
		FROM health_metrics
		WHERE metric_type = $2 AND time >= $3 AND time < $4
		GROUP BY bucket
		ORDER BY bucket`,
		interval, q.Type, q.From, q.To)
	if err != nil {
		return nil, fmt.Errorf("query metrics: %w", err)
	}
	defer rows.Close()

	var results []model.AggregatedMetric
	for rows.Next() {
		var m model.AggregatedMetric
		if err := rows.Scan(&m.Time, &m.Avg, &m.Min, &m.Max, &m.Sum, &m.Count); err != nil {
			return nil, fmt.Errorf("scan metric: %w", err)
		}
		results = append(results, m)
	}
	return results, rows.Err()
}

// QuerySleepSessions returns sleep sessions in a time range.
func (r *Repository) QuerySleepSessions(ctx context.Context, q model.TimeRangeQuery) ([]model.SleepSession, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, start_time, end_time, duration_m, stages
		FROM sleep_sessions
		WHERE start_time >= $1 AND start_time < $2
		ORDER BY start_time DESC`, q.From, q.To)
	if err != nil {
		return nil, fmt.Errorf("query sleep: %w", err)
	}
	defer rows.Close()

	var results []model.SleepSession
	for rows.Next() {
		var s model.SleepSession
		var stagesJSON []byte
		if err := rows.Scan(&s.ID, &s.StartTime, &s.EndTime, &s.DurationM, &stagesJSON); err != nil {
			return nil, fmt.Errorf("scan sleep: %w", err)
		}
		_ = json.Unmarshal(stagesJSON, &s.Stages)
		results = append(results, s)
	}
	return results, rows.Err()
}

// QueryExerciseSessions returns exercise sessions in a time range.
func (r *Repository) QueryExerciseSessions(ctx context.Context, q model.TimeRangeQuery) ([]model.ExerciseSession, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, exercise_type, start_time, end_time, duration_m, calories_kcal, distance_m, metadata
		FROM exercise_sessions
		WHERE start_time >= $1 AND start_time < $2
		ORDER BY start_time DESC`, q.From, q.To)
	if err != nil {
		return nil, fmt.Errorf("query exercise: %w", err)
	}
	defer rows.Close()

	var results []model.ExerciseSession
	for rows.Next() {
		var s model.ExerciseSession
		var metaJSON []byte
		if err := rows.Scan(&s.ID, &s.ExerciseType, &s.StartTime, &s.EndTime, &s.DurationM, &s.CaloriesKcal, &s.DistanceM, &metaJSON); err != nil {
			return nil, fmt.Errorf("scan exercise: %w", err)
		}
		if metaJSON != nil {
			_ = json.Unmarshal(metaJSON, &s.Metadata)
		}
		results = append(results, s)
	}
	return results, rows.Err()
}

// QueryNutritionRecords returns nutrition records in a time range.
func (r *Repository) QueryNutritionRecords(ctx context.Context, q model.TimeRangeQuery) ([]model.NutritionRecord, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, time, name, meal_type, calories, protein_g, fat_g, carbs_g, notes, metadata
		FROM nutrition_records
		WHERE time >= $1 AND time < $2
		ORDER BY time DESC`, q.From, q.To)
	if err != nil {
		return nil, fmt.Errorf("query nutrition: %w", err)
	}
	defer rows.Close()

	var results []model.NutritionRecord
	for rows.Next() {
		var n model.NutritionRecord
		var metaJSON []byte
		if err := rows.Scan(&n.ID, &n.Time, &n.Name, &n.MealType, &n.Calories, &n.ProteinG, &n.FatG, &n.CarbsG, &n.Notes, &metaJSON); err != nil {
			return nil, fmt.Errorf("scan nutrition: %w", err)
		}
		if metaJSON != nil {
			_ = json.Unmarshal(metaJSON, &n.Metadata)
		}
		results = append(results, n)
	}
	return results, rows.Err()
}

// QueryBodyMeasurements returns body measurements in a time range.
func (r *Repository) QueryBodyMeasurements(ctx context.Context, q model.TimeRangeQuery) ([]model.BodyMeasurement, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT time, weight_kg, body_fat_pct, lean_mass_kg, skeletal_muscle_mass_kg, body_fat_mass_kg
		FROM body_measurements
		WHERE time >= $1 AND time < $2
		ORDER BY time DESC`, q.From, q.To)
	if err != nil {
		return nil, fmt.Errorf("query body: %w", err)
	}
	defer rows.Close()

	var results []model.BodyMeasurement
	for rows.Next() {
		var m model.BodyMeasurement
		if err := rows.Scan(&m.Time, &m.WeightKg, &m.BodyFatPct, &m.LeanMassKg, &m.SkeletalMuscleMassKg, &m.BodyFatMassKg); err != nil {
			return nil, fmt.Errorf("scan body: %w", err)
		}
		results = append(results, m)
	}
	return results, rows.Err()
}

// GetDailySummary returns an aggregated summary for a given date.
func (r *Repository) GetDailySummary(ctx context.Context, date time.Time) (*model.DailySummary, error) {
	dayStart := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	dayEnd := dayStart.Add(24 * time.Hour)

	summary := &model.DailySummary{Date: dayStart.Format("2006-01-02")}

	// Steps
	var steps *float64
	err := r.pool.QueryRow(ctx,
		`SELECT SUM(value) FROM health_metrics WHERE metric_type = 'steps' AND time >= $1 AND time < $2`,
		dayStart, dayEnd).Scan(&steps)
	if err != nil {
		return nil, fmt.Errorf("summary steps: %w", err)
	}
	summary.TotalSteps = steps

	// Heart rate
	var hr *float64
	err = r.pool.QueryRow(ctx,
		`SELECT AVG(value) FROM health_metrics WHERE metric_type = 'heart_rate' AND time >= $1 AND time < $2`,
		dayStart, dayEnd).Scan(&hr)
	if err != nil {
		return nil, fmt.Errorf("summary hr: %w", err)
	}
	summary.AvgHeartRate = hr

	// Calories
	var cal *float64
	err = r.pool.QueryRow(ctx,
		`SELECT SUM(value) FROM health_metrics WHERE metric_type = 'calories' AND time >= $1 AND time < $2`,
		dayStart, dayEnd).Scan(&cal)
	if err != nil {
		return nil, fmt.Errorf("summary calories: %w", err)
	}
	summary.TotalCalories = cal

	// Distance
	var dist *float64
	err = r.pool.QueryRow(ctx,
		`SELECT SUM(value) FROM health_metrics WHERE metric_type = 'distance' AND time >= $1 AND time < $2`,
		dayStart, dayEnd).Scan(&dist)
	if err != nil {
		return nil, fmt.Errorf("summary distance: %w", err)
	}
	summary.TotalDistanceM = dist

	// SpO2
	var spo2 *float64
	err = r.pool.QueryRow(ctx,
		`SELECT AVG(value) FROM health_metrics WHERE metric_type = 'spo2' AND time >= $1 AND time < $2`,
		dayStart, dayEnd).Scan(&spo2)
	if err != nil {
		return nil, fmt.Errorf("summary spo2: %w", err)
	}
	summary.AvgSpO2 = spo2

	// Sleep (find session that started in the range or ended in the range)
	sleepSessions, err := r.QuerySleepSessions(ctx, model.TimeRangeQuery{From: dayStart.Add(-12 * time.Hour), To: dayEnd})
	if err != nil {
		return nil, fmt.Errorf("summary sleep: %w", err)
	}
	if len(sleepSessions) > 0 {
		summary.Sleep = &sleepSessions[0]
	}

	// Exercises
	exercises, err := r.QueryExerciseSessions(ctx, model.TimeRangeQuery{From: dayStart, To: dayEnd})
	if err != nil {
		return nil, fmt.Errorf("summary exercises: %w", err)
	}
	summary.Exercises = exercises

	// Body
	bodyMeasurements, err := r.QueryBodyMeasurements(ctx, model.TimeRangeQuery{From: dayStart, To: dayEnd})
	if err != nil {
		return nil, fmt.Errorf("summary body: %w", err)
	}
	if len(bodyMeasurements) > 0 {
		summary.Weight = &bodyMeasurements[0]
	}

	return summary, nil
}

// PurgeNonWebhookData truncates all tables, keeping schema intact.
// HC Webhook can resync 48h of data after purge.
func (r *Repository) PurgeNonWebhookData(ctx context.Context) (int64, error) {
	tables := []string{"health_metrics", "sleep_sessions", "exercise_sessions", "nutrition_records", "body_measurements"}
	var total int64
	for _, t := range tables {
		ct, err := r.pool.Exec(ctx, "DELETE FROM "+t)
		if err != nil {
			return total, fmt.Errorf("purge %s: %w", t, err)
		}
		total += ct.RowsAffected()
	}
	return total, nil
}
