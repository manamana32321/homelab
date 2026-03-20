package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/manamana32321/homelab/health-hub/internal/model"
)

// HC Webhook payload format from https://github.com/mcnaveen/health-connect-webhook
type hcWebhookPayload struct {
	Timestamp        string              `json:"timestamp"`
	Steps            []hcSteps           `json:"steps"`
	Sleep            []hcSleep           `json:"sleep"`
	HeartRate        []hcHeartRate       `json:"heart_rate"`
	HRV              []hcHRV             `json:"heart_rate_variability"`
	Distance         []hcDistance        `json:"distance"`
	ActiveCalories   []hcCalories        `json:"active_calories"`
	TotalCalories    []hcCalories        `json:"total_calories"`
	Weight           []hcWeight          `json:"weight"`
	OxygenSaturation []hcOxygenSat       `json:"oxygen_saturation"`
	Exercise         []hcExercise        `json:"exercise"`
	Hydration        []hcHydration       `json:"hydration"`
	Nutrition        []hcNutrition       `json:"nutrition"`
	RestingHeartRate []hcHeartRate       `json:"resting_heart_rate"`
	BloodPressure    []hcBloodPressure   `json:"blood_pressure"`
	BodyTemperature  []hcBodyTemp        `json:"body_temperature"`
	RespiratoryRate  []hcRespiratoryRate `json:"respiratory_rate"`
}

type hcSteps struct {
	Count     int64  `json:"count"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
}

type hcSleep struct {
	SessionEndTime  string         `json:"session_end_time"`
	DurationSeconds int64          `json:"duration_seconds"`
	Stages          []hcSleepStage `json:"stages"`
}

type hcSleepStage struct {
	Stage           string `json:"stage"`
	StartTime       string `json:"start_time"`
	EndTime         string `json:"end_time"`
	DurationSeconds int64  `json:"duration_seconds"`
}

type hcHeartRate struct {
	BPM  float64 `json:"bpm"`
	Time string  `json:"time"`
}

type hcHRV struct {
	RmssdMillis float64 `json:"rmssd_millis"`
	Time        string  `json:"time"`
}

type hcDistance struct {
	Meters    float64 `json:"meters"`
	StartTime string  `json:"start_time"`
	EndTime   string  `json:"end_time"`
}

type hcCalories struct {
	Calories  float64 `json:"calories"`
	StartTime string  `json:"start_time"`
	EndTime   string  `json:"end_time"`
}

type hcWeight struct {
	Kilograms float64 `json:"kilograms"`
	Time      string  `json:"time"`
}

type hcOxygenSat struct {
	Percentage float64 `json:"percentage"`
	Time       string  `json:"time"`
}

type hcExercise struct {
	Type            string `json:"type"`
	StartTime       string `json:"start_time"`
	EndTime         string `json:"end_time"`
	DurationSeconds int64  `json:"duration_seconds"`
}

type hcHydration struct {
	Liters    float64 `json:"liters"`
	StartTime string  `json:"start_time"`
	EndTime   string  `json:"end_time"`
}

type hcNutrition struct {
	Calories     *float64 `json:"calories"`
	ProteinGrams *float64 `json:"protein_grams"`
	CarbsGrams   *float64 `json:"carbs_grams"`
	FatGrams     *float64 `json:"fat_grams"`
	StartTime    string   `json:"start_time"`
	EndTime      string   `json:"end_time"`
}

type hcBloodPressure struct {
	Systolic  float64 `json:"systolic"`
	Diastolic float64 `json:"diastolic"`
	Time      string  `json:"time"`
}

type hcBodyTemp struct {
	Celsius float64 `json:"celsius"`
	Time    string  `json:"time"`
}

type hcRespiratoryRate struct {
	Rate float64 `json:"rate"`
	Time string  `json:"time"`
}

func parseTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		t, err = time.Parse(time.RFC3339, s)
		if err != nil {
			t, _ = time.Parse("2006-01-02T15:04:05Z", s)
		}
	}
	return t
}

func (h *handler) hcWebhook(w http.ResponseWriter, r *http.Request) {
	var payload hcWebhookPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json: "+err.Error())
		return
	}

	ctx := r.Context()
	result := model.IngestResult{}

	// Convert steps → metrics
	var metrics []model.Metric
	for _, s := range payload.Steps {
		metrics = append(metrics, model.Metric{
			Time: parseTime(s.EndTime), MetricType: "steps", Value: float64(s.Count), Unit: "count", Source: "hc_webhook",
		})
	}
	for _, hr := range payload.HeartRate {
		metrics = append(metrics, model.Metric{
			Time: parseTime(hr.Time), MetricType: "heart_rate", Value: hr.BPM, Unit: "bpm", Source: "hc_webhook",
		})
	}
	for _, hr := range payload.RestingHeartRate {
		metrics = append(metrics, model.Metric{
			Time: parseTime(hr.Time), MetricType: "resting_heart_rate", Value: hr.BPM, Unit: "bpm", Source: "hc_webhook",
		})
	}
	for _, hrv := range payload.HRV {
		metrics = append(metrics, model.Metric{
			Time: parseTime(hrv.Time), MetricType: "hrv", Value: hrv.RmssdMillis, Unit: "ms", Source: "hc_webhook",
		})
	}
	for _, d := range payload.Distance {
		metrics = append(metrics, model.Metric{
			Time: parseTime(d.EndTime), MetricType: "distance", Value: d.Meters, Unit: "m", Source: "hc_webhook",
		})
	}
	for _, c := range payload.ActiveCalories {
		metrics = append(metrics, model.Metric{
			Time: parseTime(c.EndTime), MetricType: "active_calories", Value: c.Calories, Unit: "kcal", Source: "hc_webhook",
		})
	}
	for _, c := range payload.TotalCalories {
		metrics = append(metrics, model.Metric{
			Time: parseTime(c.EndTime), MetricType: "calories", Value: c.Calories, Unit: "kcal", Source: "hc_webhook",
		})
	}
	for _, o := range payload.OxygenSaturation {
		metrics = append(metrics, model.Metric{
			Time: parseTime(o.Time), MetricType: "spo2", Value: o.Percentage, Unit: "percent", Source: "hc_webhook",
		})
	}
	for _, hy := range payload.Hydration {
		metrics = append(metrics, model.Metric{
			Time: parseTime(hy.EndTime), MetricType: "hydration", Value: hy.Liters * 1000, Unit: "mL", Source: "hc_webhook",
		})
	}
	for _, bp := range payload.BloodPressure {
		t := parseTime(bp.Time)
		metrics = append(metrics, model.Metric{
			Time: t, MetricType: "blood_pressure_systolic", Value: bp.Systolic, Unit: "mmHg", Source: "hc_webhook",
		})
		metrics = append(metrics, model.Metric{
			Time: t, MetricType: "blood_pressure_diastolic", Value: bp.Diastolic, Unit: "mmHg", Source: "hc_webhook",
		})
	}
	for _, bt := range payload.BodyTemperature {
		metrics = append(metrics, model.Metric{
			Time: parseTime(bt.Time), MetricType: "body_temperature", Value: bt.Celsius, Unit: "celsius", Source: "hc_webhook",
		})
	}
	for _, rr := range payload.RespiratoryRate {
		metrics = append(metrics, model.Metric{
			Time: parseTime(rr.Time), MetricType: "respiratory_rate", Value: rr.Rate, Unit: "bpm", Source: "hc_webhook",
		})
	}

	n, err := h.repo.InsertMetrics(ctx, metrics)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "insert metrics: "+err.Error())
		return
	}
	result.Metrics = n

	// Convert sleep
	var sleepSessions []model.SleepSession
	for _, s := range payload.Sleep {
		endTime := parseTime(s.SessionEndTime)
		startTime := endTime.Add(-time.Duration(s.DurationSeconds) * time.Second)
		var stages []model.SleepStage
		for _, st := range s.Stages {
			stages = append(stages, model.SleepStage{
				Stage:     st.Stage,
				StartTime: parseTime(st.StartTime),
				EndTime:   parseTime(st.EndTime),
				DurationM: int(st.DurationSeconds / 60),
			})
		}
		sleepSessions = append(sleepSessions, model.SleepSession{
			StartTime: startTime,
			EndTime:   endTime,
			DurationM: int(s.DurationSeconds / 60),
			Stages:    stages,
		})
	}
	n, err = h.repo.InsertSleepSessions(ctx, sleepSessions)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "insert sleep: "+err.Error())
		return
	}
	result.Sleep = n

	// Convert exercises
	var exercises []model.ExerciseSession
	for _, e := range payload.Exercise {
		exercises = append(exercises, model.ExerciseSession{
			ExerciseType: e.Type,
			StartTime:    parseTime(e.StartTime),
			EndTime:      parseTime(e.EndTime),
			DurationM:    int(e.DurationSeconds / 60),
		})
	}
	n, err = h.repo.InsertExerciseSessions(ctx, exercises)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "insert exercises: "+err.Error())
		return
	}
	result.Exercises = n

	// Convert nutrition
	var nutrition []model.NutritionRecord
	for _, nu := range payload.Nutrition {
		nutrition = append(nutrition, model.NutritionRecord{
			Time:     parseTime(nu.StartTime),
			Calories: nu.Calories,
			ProteinG: nu.ProteinGrams,
			CarbsG:   nu.CarbsGrams,
			FatG:     nu.FatGrams,
		})
	}
	n, err = h.repo.InsertNutritionRecords(ctx, nutrition)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "insert nutrition: "+err.Error())
		return
	}
	result.Nutrition = n

	// Convert weight → body measurements
	var body []model.BodyMeasurement
	for _, wt := range payload.Weight {
		kg := wt.Kilograms
		body = append(body, model.BodyMeasurement{
			Time:     parseTime(wt.Time),
			WeightKg: &kg,
		})
	}
	n, err = h.repo.InsertBodyMeasurements(ctx, body)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "insert body: "+err.Error())
		return
	}
	result.Body = n

	writeJSON(w, http.StatusOK, result)
}
