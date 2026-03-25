package api

import (
	"context"
	"time"

	"github.com/manamana32321/homelab/health-hub/internal/model"
)

// DataRepository defines the interface for data access used by handlers.
type DataRepository interface {
	Ping(ctx context.Context) error
	InsertMetrics(ctx context.Context, metrics []model.Metric) (int, error)
	InsertSleepSessions(ctx context.Context, sessions []model.SleepSession) (int, error)
	InsertExerciseSessions(ctx context.Context, sessions []model.ExerciseSession) (int, error)
	InsertNutritionRecords(ctx context.Context, records []model.NutritionRecord) (int, error)
	InsertNutritionRecord(ctx context.Context, n model.NutritionRecord) (int64, error)
	UpdateNutritionRecord(ctx context.Context, n model.NutritionRecord) error
	DeleteNutritionRecord(ctx context.Context, id int64) error
	GetNutritionRecord(ctx context.Context, id int64) (*model.NutritionRecord, error)
	QueryNutritionByType(ctx context.Context, q model.TimeRangeQuery, mealType string) ([]model.NutritionRecord, error)
	InsertBodyMeasurements(ctx context.Context, measurements []model.BodyMeasurement) (int, error)
	InsertBodyMeasurement(ctx context.Context, m model.BodyMeasurement) error
	InsertHealthNote(ctx context.Context, n model.HealthNote) (int64, error)
	QueryHealthNotes(ctx context.Context, q model.TimeRangeQuery, category string) ([]model.HealthNote, error)
	QueryMetrics(ctx context.Context, q model.MetricsQuery) ([]model.AggregatedMetric, error)
	QuerySleepSessions(ctx context.Context, q model.TimeRangeQuery) ([]model.SleepSession, error)
	QueryExerciseSessions(ctx context.Context, q model.TimeRangeQuery) ([]model.ExerciseSession, error)
	QueryNutritionRecords(ctx context.Context, q model.TimeRangeQuery) ([]model.NutritionRecord, error)
	QueryBodyMeasurements(ctx context.Context, q model.TimeRangeQuery) ([]model.BodyMeasurement, error)
	GetDailySummary(ctx context.Context, date time.Time) (*model.DailySummary, error)
	PurgeNonWebhookData(ctx context.Context) (int64, error)
}
