package model

import "time"

// Metric represents a single time-series health metric (steps, heart_rate, spo2, etc.)
type Metric struct {
	Time       time.Time              `json:"time"`
	MetricType string                 `json:"type"`
	Value      float64                `json:"value"`
	Unit       string                 `json:"unit"`
	Source     string                 `json:"source,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// SleepSession represents a complete sleep session with stages.
type SleepSession struct {
	ID        int64       `json:"id,omitempty"`
	StartTime time.Time   `json:"start_time"`
	EndTime   time.Time   `json:"end_time"`
	DurationM int         `json:"duration_m"`
	Stages    []SleepStage `json:"stages"`
}

type SleepStage struct {
	Stage     string    `json:"stage"` // deep, light, rem, awake
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	DurationM int       `json:"duration_m"`
}

// ExerciseSession represents a workout session.
type ExerciseSession struct {
	ID           int64                  `json:"id,omitempty"`
	ExerciseType string                 `json:"exercise_type"`
	StartTime    time.Time              `json:"start_time"`
	EndTime      time.Time              `json:"end_time"`
	DurationM    int                    `json:"duration_m"`
	CaloriesKcal *float64               `json:"calories_kcal,omitempty"`
	DistanceM    *float64               `json:"distance_m,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// NutritionRecord represents a food/nutrition entry.
type NutritionRecord struct {
	ID       int64                  `json:"id,omitempty"`
	Time     time.Time              `json:"time"`
	MealType *string                `json:"meal_type,omitempty"`
	Calories *float64               `json:"calories,omitempty"`
	ProteinG *float64               `json:"protein_g,omitempty"`
	FatG     *float64               `json:"fat_g,omitempty"`
	CarbsG   *float64               `json:"carbs_g,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// BodyMeasurement represents weight and body composition.
type BodyMeasurement struct {
	Time       time.Time `json:"time"`
	WeightKg   *float64  `json:"weight_kg,omitempty"`
	BodyFatPct *float64  `json:"body_fat_pct,omitempty"`
	LeanMassKg *float64  `json:"lean_mass_kg,omitempty"`
}

// IngestPayload is the request body from Tasker.
type IngestPayload struct {
	Timestamp  time.Time         `json:"timestamp"`
	Metrics    []Metric          `json:"metrics"`
	Sleep      []SleepSession    `json:"sleep_sessions"`
	Exercises  []ExerciseSession `json:"exercises"`
	Nutrition  []NutritionRecord `json:"nutrition"`
	Body       []BodyMeasurement `json:"body"`
}

// IngestResult is the response after ingesting data.
type IngestResult struct {
	Metrics   int `json:"metrics_inserted"`
	Sleep     int `json:"sleep_inserted"`
	Exercises int `json:"exercises_inserted"`
	Nutrition int `json:"nutrition_inserted"`
	Body      int `json:"body_inserted"`
}

// MetricsQuery parameters for querying time-series metrics.
type MetricsQuery struct {
	Type     string
	From     time.Time
	To       time.Time
	Interval string // 1m, 5m, 15m, 1h, 1d
}

// TimeRangeQuery for querying session-based data.
type TimeRangeQuery struct {
	From time.Time
	To   time.Time
}

// AggregatedMetric is a single aggregated data point.
type AggregatedMetric struct {
	Time  time.Time `json:"time"`
	Avg   float64   `json:"avg"`
	Min   float64   `json:"min"`
	Max   float64   `json:"max"`
	Sum   float64   `json:"sum"`
	Count int       `json:"count"`
}

// DailySummary is the response for a single day's health overview.
type DailySummary struct {
	Date           string           `json:"date"`
	TotalSteps     *float64         `json:"total_steps,omitempty"`
	AvgHeartRate   *float64         `json:"avg_heart_rate,omitempty"`
	TotalCalories  *float64         `json:"total_calories,omitempty"`
	TotalDistanceM *float64         `json:"total_distance_m,omitempty"`
	Sleep          *SleepSession    `json:"sleep,omitempty"`
	Exercises      []ExerciseSession `json:"exercises,omitempty"`
	AvgSpO2        *float64         `json:"avg_spo2,omitempty"`
	Weight         *BodyMeasurement `json:"weight,omitempty"`
}
