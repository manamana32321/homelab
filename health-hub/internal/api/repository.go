package api

import (
	"context"
	"time"

	"github.com/manamana32321/homelab/health-hub/internal/model"
)

// DataRepository defines the interface for data access used by handlers.
// This allows for mocking in tests without requiring a real database.
type DataRepository interface {
	Ping(ctx context.Context) error
	InsertMetrics(ctx context.Context, metrics []model.Metric) (int, error)
	InsertSleepSessions(ctx context.Context, sessions []model.SleepSession) (int, error)
	InsertExerciseSessions(ctx context.Context, sessions []model.ExerciseSession) (int, error)
	InsertNutritionRecords(ctx context.Context, records []model.NutritionRecord) (int, error)
	InsertBodyMeasurements(ctx context.Context, measurements []model.BodyMeasurement) (int, error)
	QueryMetrics(ctx context.Context, q model.MetricsQuery) ([]model.AggregatedMetric, error)
	QuerySleepSessions(ctx context.Context, q model.TimeRangeQuery) ([]model.SleepSession, error)
	QueryExerciseSessions(ctx context.Context, q model.TimeRangeQuery) ([]model.ExerciseSession, error)
	QueryNutritionRecords(ctx context.Context, q model.TimeRangeQuery) ([]model.NutritionRecord, error)
	QueryBodyMeasurements(ctx context.Context, q model.TimeRangeQuery) ([]model.BodyMeasurement, error)
	GetDailySummary(ctx context.Context, date time.Time) (*model.DailySummary, error)
	PurgeNonWebhookData(ctx context.Context) (int64, error)
}
