package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/manamana32321/homelab/health-hub/internal/model"
)

// mockRepo implements DataRepository for testing.
type mockRepo struct {
	metrics    []model.Metric
	sleep      []model.SleepSession
	exercises  []model.ExerciseSession
	nutrition  []model.NutritionRecord
	body       []model.BodyMeasurement
	pingErr    error
	insertErr  error
}

func (m *mockRepo) Ping(_ context.Context) error { return m.pingErr }

func (m *mockRepo) InsertMetrics(_ context.Context, metrics []model.Metric) (int, error) {
	if m.insertErr != nil {
		return 0, m.insertErr
	}
	m.metrics = append(m.metrics, metrics...)
	return len(metrics), nil
}

func (m *mockRepo) InsertSleepSessions(_ context.Context, sessions []model.SleepSession) (int, error) {
	if m.insertErr != nil {
		return 0, m.insertErr
	}
	m.sleep = append(m.sleep, sessions...)
	return len(sessions), nil
}

func (m *mockRepo) InsertExerciseSessions(_ context.Context, sessions []model.ExerciseSession) (int, error) {
	if m.insertErr != nil {
		return 0, m.insertErr
	}
	m.exercises = append(m.exercises, sessions...)
	return len(sessions), nil
}

func (m *mockRepo) InsertNutritionRecords(_ context.Context, records []model.NutritionRecord) (int, error) {
	if m.insertErr != nil {
		return 0, m.insertErr
	}
	m.nutrition = append(m.nutrition, records...)
	return len(records), nil
}

func (m *mockRepo) InsertBodyMeasurements(_ context.Context, measurements []model.BodyMeasurement) (int, error) {
	if m.insertErr != nil {
		return 0, m.insertErr
	}
	m.body = append(m.body, measurements...)
	return len(measurements), nil
}

func (m *mockRepo) QueryMetrics(_ context.Context, q model.MetricsQuery) ([]model.AggregatedMetric, error) {
	var results []model.AggregatedMetric
	for _, metric := range m.metrics {
		if metric.MetricType == q.Type && !metric.Time.Before(q.From) && metric.Time.Before(q.To) {
			results = append(results, model.AggregatedMetric{
				Time:  metric.Time,
				Avg:   metric.Value,
				Min:   metric.Value,
				Max:   metric.Value,
				Sum:   metric.Value,
				Count: 1,
			})
		}
	}
	return results, nil
}

func (m *mockRepo) QuerySleepSessions(_ context.Context, q model.TimeRangeQuery) ([]model.SleepSession, error) {
	var results []model.SleepSession
	for _, s := range m.sleep {
		if !s.StartTime.Before(q.From) && s.StartTime.Before(q.To) {
			results = append(results, s)
		}
	}
	return results, nil
}

func (m *mockRepo) QueryExerciseSessions(_ context.Context, q model.TimeRangeQuery) ([]model.ExerciseSession, error) {
	var results []model.ExerciseSession
	for _, s := range m.exercises {
		if !s.StartTime.Before(q.From) && s.StartTime.Before(q.To) {
			results = append(results, s)
		}
	}
	return results, nil
}

func (m *mockRepo) QueryNutritionRecords(_ context.Context, q model.TimeRangeQuery) ([]model.NutritionRecord, error) {
	var results []model.NutritionRecord
	for _, n := range m.nutrition {
		if !n.Time.Before(q.From) && n.Time.Before(q.To) {
			results = append(results, n)
		}
	}
	return results, nil
}

func (m *mockRepo) QueryBodyMeasurements(_ context.Context, q model.TimeRangeQuery) ([]model.BodyMeasurement, error) {
	var results []model.BodyMeasurement
	for _, b := range m.body {
		if !b.Time.Before(q.From) && b.Time.Before(q.To) {
			results = append(results, b)
		}
	}
	return results, nil
}

func (m *mockRepo) GetDailySummary(_ context.Context, date time.Time) (*model.DailySummary, error) {
	dayStart := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	summary := &model.DailySummary{Date: dayStart.Format("2006-01-02")}

	var totalSteps float64
	for _, metric := range m.metrics {
		if metric.MetricType == "steps" && metric.Time.Format("2006-01-02") == summary.Date {
			totalSteps += metric.Value
		}
	}
	if totalSteps > 0 {
		summary.TotalSteps = &totalSteps
	}

	return summary, nil
}

func newTestServer(repo DataRepository, token string) *httptest.Server {
	return httptest.NewServer(NewRouter(repo, token))
}

func TestHealthz(t *testing.T) {
	repo := &mockRepo{}
	srv := newTestServer(repo, "")
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var body map[string]string
	json.NewDecoder(resp.Body).Decode(&body)
	if body["status"] != "ok" {
		t.Errorf("expected status ok, got %s", body["status"])
	}
}

func TestHealthzDBDown(t *testing.T) {
	repo := &mockRepo{pingErr: fmt.Errorf("connection refused")}
	srv := newTestServer(repo, "")
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", resp.StatusCode)
	}
}

func TestAuthMiddleware(t *testing.T) {
	repo := &mockRepo{}
	srv := newTestServer(repo, "test-token-123")
	defer srv.Close()

	// No auth header → 401
	resp, err := http.Get(srv.URL + "/api/v1/metrics?type=steps")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 without auth, got %d", resp.StatusCode)
	}

	// Wrong token → 401
	req, _ := http.NewRequest("GET", srv.URL+"/api/v1/metrics?type=steps", nil)
	req.Header.Set("Authorization", "Bearer wrong-token")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 with wrong token, got %d", resp.StatusCode)
	}

	// Correct token → 200
	req, _ = http.NewRequest("GET", srv.URL+"/api/v1/metrics?type=steps", nil)
	req.Header.Set("Authorization", "Bearer test-token-123")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 with correct token, got %d", resp.StatusCode)
	}

	// Healthz should bypass auth
	resp, err = http.Get(srv.URL + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 for healthz without auth, got %d", resp.StatusCode)
	}
}

func TestIngest(t *testing.T) {
	repo := &mockRepo{}
	srv := newTestServer(repo, "")
	defer srv.Close()

	now := time.Now().Truncate(time.Second)
	payload := model.IngestPayload{
		Timestamp: now,
		Metrics: []model.Metric{
			{Time: now, MetricType: "steps", Value: 150, Unit: "count"},
			{Time: now, MetricType: "heart_rate", Value: 72, Unit: "bpm"},
			{Time: now, MetricType: "spo2", Value: 97, Unit: "percent"},
		},
		Sleep: []model.SleepSession{
			{
				StartTime: now.Add(-8 * time.Hour),
				EndTime:   now,
				DurationM: 480,
				Stages: []model.SleepStage{
					{Stage: "deep", StartTime: now.Add(-8 * time.Hour), EndTime: now.Add(-6 * time.Hour), DurationM: 120},
					{Stage: "light", StartTime: now.Add(-6 * time.Hour), EndTime: now.Add(-4 * time.Hour), DurationM: 120},
				},
			},
		},
		Exercises: []model.ExerciseSession{
			{ExerciseType: "running", StartTime: now.Add(-1 * time.Hour), EndTime: now, DurationM: 60, CaloriesKcal: floatPtr(300)},
		},
		Nutrition: []model.NutritionRecord{
			{Time: now, MealType: strPtr("lunch"), Calories: floatPtr(700), ProteinG: floatPtr(30)},
		},
		Body: []model.BodyMeasurement{
			{Time: now, WeightKg: floatPtr(75.5), BodyFatPct: floatPtr(18.2)},
		},
	}

	body, _ := json.Marshal(payload)
	resp, err := http.Post(srv.URL+"/api/v1/ingest", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result model.IngestResult
	json.NewDecoder(resp.Body).Decode(&result)

	if result.Metrics != 3 {
		t.Errorf("expected 3 metrics inserted, got %d", result.Metrics)
	}
	if result.Sleep != 1 {
		t.Errorf("expected 1 sleep inserted, got %d", result.Sleep)
	}
	if result.Exercises != 1 {
		t.Errorf("expected 1 exercise inserted, got %d", result.Exercises)
	}
	if result.Nutrition != 1 {
		t.Errorf("expected 1 nutrition inserted, got %d", result.Nutrition)
	}
	if result.Body != 1 {
		t.Errorf("expected 1 body inserted, got %d", result.Body)
	}

	// Verify data was stored
	if len(repo.metrics) != 3 {
		t.Errorf("expected 3 metrics in repo, got %d", len(repo.metrics))
	}
	if len(repo.sleep) != 1 {
		t.Errorf("expected 1 sleep in repo, got %d", len(repo.sleep))
	}
}

func TestIngestInvalidJSON(t *testing.T) {
	repo := &mockRepo{}
	srv := newTestServer(repo, "")
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/api/v1/ingest", "application/json", bytes.NewReader([]byte("invalid")))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid json, got %d", resp.StatusCode)
	}
}

func TestIngestEmpty(t *testing.T) {
	repo := &mockRepo{}
	srv := newTestServer(repo, "")
	defer srv.Close()

	payload := model.IngestPayload{Timestamp: time.Now()}
	body, _ := json.Marshal(payload)
	resp, err := http.Post(srv.URL+"/api/v1/ingest", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 for empty ingest, got %d", resp.StatusCode)
	}

	var result model.IngestResult
	json.NewDecoder(resp.Body).Decode(&result)
	if result.Metrics != 0 || result.Sleep != 0 {
		t.Errorf("expected all zeros for empty payload, got %+v", result)
	}
}

func TestQueryMetrics(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	repo := &mockRepo{
		metrics: []model.Metric{
			{Time: now.Add(-1 * time.Hour), MetricType: "steps", Value: 500, Unit: "count"},
			{Time: now.Add(-30 * time.Minute), MetricType: "steps", Value: 300, Unit: "count"},
			{Time: now, MetricType: "heart_rate", Value: 72, Unit: "bpm"},
		},
	}
	srv := newTestServer(repo, "")
	defer srv.Close()

	// Query steps
	resp, err := http.Get(buildURL(srv.URL, "/api/v1/metrics", map[string]string{
		"type": "steps",
		"from": now.Add(-2 * time.Hour).Format(time.RFC3339),
		"to":   now.Add(1 * time.Hour).Format(time.RFC3339),
	}))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var results []model.AggregatedMetric
	json.NewDecoder(resp.Body).Decode(&results)
	if len(results) != 2 {
		t.Errorf("expected 2 step results, got %d", len(results))
	}
}

func TestQueryMetricsMissingType(t *testing.T) {
	repo := &mockRepo{}
	srv := newTestServer(repo, "")
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/v1/metrics")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for missing type, got %d", resp.StatusCode)
	}
}

func TestQuerySleep(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	repo := &mockRepo{
		sleep: []model.SleepSession{
			{ID: 1, StartTime: now.Add(-8 * time.Hour), EndTime: now, DurationM: 480, Stages: []model.SleepStage{{Stage: "deep", DurationM: 120}}},
		},
	}
	srv := newTestServer(repo, "")
	defer srv.Close()

	resp, err := http.Get(buildURL(srv.URL, "/api/v1/sleep", map[string]string{
		"from": now.Add(-24 * time.Hour).Format(time.RFC3339),
		"to":   now.Add(1 * time.Hour).Format(time.RFC3339),
	}))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var results []model.SleepSession
	json.NewDecoder(resp.Body).Decode(&results)
	if len(results) != 1 {
		t.Errorf("expected 1 sleep session, got %d", len(results))
	}
	if results[0].DurationM != 480 {
		t.Errorf("expected 480 min, got %d", results[0].DurationM)
	}
}

func TestQueryExercises(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	cal := 300.0
	repo := &mockRepo{
		exercises: []model.ExerciseSession{
			{ID: 1, ExerciseType: "running", StartTime: now.Add(-1 * time.Hour), EndTime: now, DurationM: 60, CaloriesKcal: &cal},
		},
	}
	srv := newTestServer(repo, "")
	defer srv.Close()

	resp, err := http.Get(buildURL(srv.URL, "/api/v1/exercises", map[string]string{
		"from": now.Add(-2 * time.Hour).Format(time.RFC3339),
		"to":   now.Add(1 * time.Hour).Format(time.RFC3339),
	}))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var results []model.ExerciseSession
	json.NewDecoder(resp.Body).Decode(&results)
	if len(results) != 1 {
		t.Errorf("expected 1 exercise, got %d", len(results))
	}
}

func TestQueryNutrition(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	repo := &mockRepo{
		nutrition: []model.NutritionRecord{
			{ID: 1, Time: now, MealType: strPtr("lunch"), Calories: floatPtr(700)},
		},
	}
	srv := newTestServer(repo, "")
	defer srv.Close()

	resp, err := http.Get(buildURL(srv.URL, "/api/v1/nutrition", map[string]string{
		"from": now.Add(-1 * time.Hour).Format(time.RFC3339),
		"to":   now.Add(1 * time.Hour).Format(time.RFC3339),
	}))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var results []model.NutritionRecord
	json.NewDecoder(resp.Body).Decode(&results)
	if len(results) != 1 {
		t.Errorf("expected 1 nutrition, got %d", len(results))
	}
}

func TestQueryBody(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	repo := &mockRepo{
		body: []model.BodyMeasurement{
			{Time: now, WeightKg: floatPtr(75.5), BodyFatPct: floatPtr(18.2)},
		},
	}
	srv := newTestServer(repo, "")
	defer srv.Close()

	resp, err := http.Get(buildURL(srv.URL, "/api/v1/body", map[string]string{
		"from": now.Add(-1 * time.Hour).Format(time.RFC3339),
		"to":   now.Add(1 * time.Hour).Format(time.RFC3339),
	}))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var results []model.BodyMeasurement
	json.NewDecoder(resp.Body).Decode(&results)
	if len(results) != 1 {
		t.Errorf("expected 1 body measurement, got %d", len(results))
	}
	if *results[0].WeightKg != 75.5 {
		t.Errorf("expected weight 75.5, got %f", *results[0].WeightKg)
	}
}

func TestSummary(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	today := now.Format("2006-01-02")
	repo := &mockRepo{
		metrics: []model.Metric{
			{Time: now, MetricType: "steps", Value: 5000, Unit: "count"},
			{Time: now, MetricType: "steps", Value: 3000, Unit: "count"},
		},
	}
	srv := newTestServer(repo, "")
	defer srv.Close()

	resp, err := http.Get(fmt.Sprintf("%s/api/v1/summary?date=%s", srv.URL, today))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var summary model.DailySummary
	json.NewDecoder(resp.Body).Decode(&summary)
	if summary.Date != today {
		t.Errorf("expected date %s, got %s", today, summary.Date)
	}
	if summary.TotalSteps == nil || *summary.TotalSteps != 8000 {
		t.Errorf("expected total steps 8000, got %v", summary.TotalSteps)
	}
}

func TestSummaryInvalidDate(t *testing.T) {
	repo := &mockRepo{}
	srv := newTestServer(repo, "")
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/v1/summary?date=not-a-date")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestQueryDefaultTimeRange(t *testing.T) {
	repo := &mockRepo{}
	srv := newTestServer(repo, "")
	defer srv.Close()

	// No from/to → should use defaults (last 7 days)
	resp, err := http.Get(srv.URL + "/api/v1/metrics?type=steps")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 with default range, got %d", resp.StatusCode)
	}
}

func TestQueryDateFormat(t *testing.T) {
	repo := &mockRepo{}
	srv := newTestServer(repo, "")
	defer srv.Close()

	// YYYY-MM-DD format
	resp, err := http.Get(srv.URL + "/api/v1/metrics?type=steps&from=2026-03-01&to=2026-03-20")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 with date format, got %d", resp.StatusCode)
	}
}

func TestEmptyResults(t *testing.T) {
	repo := &mockRepo{}
	srv := newTestServer(repo, "")
	defer srv.Close()

	endpoints := []string{
		"/api/v1/metrics?type=steps",
		"/api/v1/sleep",
		"/api/v1/exercises",
		"/api/v1/nutrition",
		"/api/v1/body",
	}

	for _, ep := range endpoints {
		resp, err := http.Get(srv.URL + ep)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("%s: expected 200, got %d", ep, resp.StatusCode)
		}

		var raw json.RawMessage
		json.NewDecoder(resp.Body).Decode(&raw)
		if string(raw) == "null" {
			t.Errorf("%s: expected empty array [], got null", ep)
		}
	}
}

func floatPtr(f float64) *float64 { return &f }
func strPtr(s string) *string     { return &s }

// buildURL builds a URL with properly encoded query parameters.
func buildURL(base, path string, params map[string]string) string {
	u, _ := url.Parse(base + path)
	q := u.Query()
	for k, v := range params {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()
	return u.String()
}
