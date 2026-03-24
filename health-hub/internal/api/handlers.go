package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/manamana32321/homelab/health-hub/internal/model"
)

type handler struct {
	repo DataRepository
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func (h *handler) healthz(w http.ResponseWriter, r *http.Request) {
	if err := h.repo.Ping(r.Context()); err != nil {
		writeError(w, http.StatusServiceUnavailable, "database unreachable")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *handler) ingest(w http.ResponseWriter, r *http.Request) {
	var payload model.IngestPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json: "+err.Error())
		return
	}

	ctx := r.Context()
	result := model.IngestResult{}

	n, err := h.repo.InsertMetrics(ctx, payload.Metrics)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "insert metrics: "+err.Error())
		return
	}
	result.Metrics = n

	n, err = h.repo.InsertSleepSessions(ctx, payload.Sleep)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "insert sleep: "+err.Error())
		return
	}
	result.Sleep = n

	n, err = h.repo.InsertExerciseSessions(ctx, payload.Exercises)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "insert exercises: "+err.Error())
		return
	}
	result.Exercises = n

	n, err = h.repo.InsertNutritionRecords(ctx, payload.Nutrition)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "insert nutrition: "+err.Error())
		return
	}
	result.Nutrition = n

	n, err = h.repo.InsertBodyMeasurements(ctx, payload.Body)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "insert body: "+err.Error())
		return
	}
	result.Body = n

	writeJSON(w, http.StatusOK, result)
}

func parseTimeRange(r *http.Request) (from, to time.Time, err error) {
	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")

	if fromStr == "" {
		from = time.Now().AddDate(0, 0, -7) // default: last 7 days
	} else {
		from, err = time.Parse(time.RFC3339, fromStr)
		if err != nil {
			from, err = time.Parse("2006-01-02", fromStr)
			if err != nil {
				return time.Time{}, time.Time{}, err
			}
		}
	}

	if toStr == "" {
		to = time.Now()
	} else {
		to, err = time.Parse(time.RFC3339, toStr)
		if err != nil {
			to, err = time.Parse("2006-01-02", toStr)
			if err != nil {
				return time.Time{}, time.Time{}, err
			}
			to = to.Add(24 * time.Hour) // end of day
		}
	}

	return from, to, nil
}

func (h *handler) queryMetrics(w http.ResponseWriter, r *http.Request) {
	metricType := r.URL.Query().Get("type")
	if metricType == "" {
		writeError(w, http.StatusBadRequest, "query parameter 'type' is required")
		return
	}

	from, to, err := parseTimeRange(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid time format: "+err.Error())
		return
	}

	interval := r.URL.Query().Get("interval")
	if interval == "" {
		interval = "1h"
	}

	results, err := h.repo.QueryMetrics(r.Context(), model.MetricsQuery{
		Type:     metricType,
		From:     from,
		To:       to,
		Interval: interval,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if results == nil {
		results = []model.AggregatedMetric{}
	}
	writeJSON(w, http.StatusOK, results)
}

func (h *handler) querySleep(w http.ResponseWriter, r *http.Request) {
	from, to, err := parseTimeRange(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid time format: "+err.Error())
		return
	}

	results, err := h.repo.QuerySleepSessions(r.Context(), model.TimeRangeQuery{From: from, To: to})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if results == nil {
		results = []model.SleepSession{}
	}
	writeJSON(w, http.StatusOK, results)
}

func (h *handler) queryExercises(w http.ResponseWriter, r *http.Request) {
	from, to, err := parseTimeRange(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid time format: "+err.Error())
		return
	}

	results, err := h.repo.QueryExerciseSessions(r.Context(), model.TimeRangeQuery{From: from, To: to})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if results == nil {
		results = []model.ExerciseSession{}
	}
	writeJSON(w, http.StatusOK, results)
}

func (h *handler) queryNutrition(w http.ResponseWriter, r *http.Request) {
	from, to, err := parseTimeRange(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid time format: "+err.Error())
		return
	}

	results, err := h.repo.QueryNutritionRecords(r.Context(), model.TimeRangeQuery{From: from, To: to})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if results == nil {
		results = []model.NutritionRecord{}
	}
	writeJSON(w, http.StatusOK, results)
}

func (h *handler) queryBody(w http.ResponseWriter, r *http.Request) {
	from, to, err := parseTimeRange(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid time format: "+err.Error())
		return
	}

	results, err := h.repo.QueryBodyMeasurements(r.Context(), model.TimeRangeQuery{From: from, To: to})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if results == nil {
		results = []model.BodyMeasurement{}
	}
	writeJSON(w, http.StatusOK, results)
}

func (h *handler) summary(w http.ResponseWriter, r *http.Request) {
	dateStr := r.URL.Query().Get("date")
	var date time.Time
	if dateStr == "" {
		date = time.Now()
	} else {
		var err error
		date, err = time.Parse("2006-01-02", dateStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid date format, use YYYY-MM-DD")
			return
		}
	}

	result, err := h.repo.GetDailySummary(r.Context(), date)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *handler) addMeal(w http.ResponseWriter, r *http.Request) {
	var meal model.NutritionRecord
	if err := json.NewDecoder(r.Body).Decode(&meal); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json: "+err.Error())
		return
	}

	if meal.Time.IsZero() {
		meal.Time = time.Now()
	}

	n, err := h.repo.InsertNutritionRecords(r.Context(), []model.NutritionRecord{meal})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{"inserted": n, "meal": meal})
}

func (h *handler) purgeData(w http.ResponseWriter, r *http.Request) {
	deleted, err := h.repo.PurgeNonWebhookData(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]int64{"deleted": deleted})
}
