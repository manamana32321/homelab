package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Metric struct {
	Time       time.Time `json:"time"`
	MetricType string    `json:"type"`
	Value      float64   `json:"value"`
	Unit       string    `json:"unit"`
	Source     string    `json:"source"`
}

type SleepSession struct {
	StartTime time.Time    `json:"start_time"`
	EndTime   time.Time    `json:"end_time"`
	DurationM int          `json:"duration_m"`
	Stages    []SleepStage `json:"stages"`
}

type SleepStage struct {
	Stage     string    `json:"stage"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	DurationM int       `json:"duration_m"`
}

type ExerciseSession struct {
	ExerciseType string   `json:"exercise_type"`
	StartTime    time.Time `json:"start_time"`
	EndTime      time.Time `json:"end_time"`
	DurationM    int       `json:"duration_m"`
	CaloriesKcal *float64  `json:"calories_kcal,omitempty"`
	DistanceM    *float64  `json:"distance_m,omitempty"`
}

type IngestPayload struct {
	Timestamp time.Time         `json:"timestamp"`
	Metrics   []Metric          `json:"metrics"`
	Sleep     []SleepSession    `json:"sleep_sessions"`
	Exercises []ExerciseSession `json:"exercises"`
}

const batchSize = 500

var (
	apiURL string
	token  string
	limit  int
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: migrate <samsung-health-jsons-dir> <api-url> [api-token] [--limit N]\n")
		os.Exit(1)
	}

	dir := os.Args[1]
	apiURL = os.Args[2]
	for i := 3; i < len(os.Args); i++ {
		if os.Args[i] == "--limit" && i+1 < len(os.Args) {
			fmt.Sscanf(os.Args[i+1], "%d", &limit)
			i++
		} else if token == "" {
			token = os.Args[i]
		}
	}

	log.Printf("Samsung Health JSON migration → %s (limit=%d)", apiURL, limit)

	migrateHeartRate(dir)
	migrateSteps(dir)
	migrateSpO2(dir)
	migrateExercise(dir)
	migrateSleep(dir)

	log.Println("Done!")
}

// ── Heart Rate ──────────────────────────────────────────────

type shHeartRate struct {
	HeartRate float64 `json:"heart_rate"`
	StartTime int64   `json:"start_time"`
}

func migrateHeartRate(dir string) {
	records := collectJSON[shHeartRate](dir, "com.samsung.shealth.tracker.heart_rate")
	var metrics []Metric
	for _, r := range records {
		if r.HeartRate <= 0 || r.HeartRate > 300 || r.StartTime == 0 {
			continue
		}
		metrics = append(metrics, Metric{
			Time: time.UnixMilli(r.StartTime), MetricType: "heart_rate",
			Value: r.HeartRate, Unit: "bpm", Source: "samsung_export",
		})
	}
	log.Printf("HeartRate: %d raw → %d valid", len(records), len(metrics))
	sendMetrics(metrics)
}

// ── Steps (pedometer_day_summary binning_data) ──────────────

type shPedometerBin struct {
	MStepCount int     `json:"mStepCount"`
	MCalorie   float64 `json:"mCalorie"`
	MDistance   float64 `json:"mDistance"`
	MStartTime int64   `json:"mStartTime"`
}

func migrateSteps(dir string) {
	records := collectJSON[shPedometerBin](dir, "com.samsung.shealth.tracker.pedometer_day_summary")
	var metrics []Metric
	for _, r := range records {
		if r.MStepCount <= 0 || r.MStartTime == 0 {
			continue
		}
		t := time.UnixMilli(r.MStartTime)
		metrics = append(metrics, Metric{
			Time: t, MetricType: "steps", Value: float64(r.MStepCount), Unit: "count", Source: "samsung_export",
		})
		if r.MDistance > 0 {
			metrics = append(metrics, Metric{
				Time: t, MetricType: "distance", Value: r.MDistance, Unit: "m", Source: "samsung_export",
			})
		}
		if r.MCalorie > 0 {
			metrics = append(metrics, Metric{
				Time: t, MetricType: "calories", Value: r.MCalorie, Unit: "kcal", Source: "samsung_export",
			})
		}
	}
	log.Printf("Steps: %d raw → %d metric records (steps+dist+cal)", len(records), len(metrics))
	sendMetrics(metrics)
}

// ── SpO2 ────────────────────────────────────────────────────

type shSpO2 struct {
	SpO2      float64 `json:"spo2"`
	StartTime int64   `json:"start_time"`
}

func migrateSpO2(dir string) {
	records := collectJSON[shSpO2](dir, "com.samsung.shealth.tracker.oxygen_saturation")
	var metrics []Metric
	for _, r := range records {
		if r.SpO2 <= 0 || r.SpO2 > 100 || r.StartTime == 0 {
			continue
		}
		metrics = append(metrics, Metric{
			Time: time.UnixMilli(r.StartTime), MetricType: "spo2",
			Value: r.SpO2, Unit: "percent", Source: "samsung_export",
		})
	}
	log.Printf("SpO2: %d raw → %d valid", len(records), len(metrics))
	sendMetrics(metrics)
}

// ── Exercise ────────────────────────────────────────────────
// Each exercise UUID has a .live_data.json or .live_data_internal.json
// with per-minute entries containing start_time, calorie, distance, speed.
// We aggregate each UUID's entries into one ExerciseSession.

type shExerciseLive struct {
	Calorie   float64 `json:"calorie"`
	Distance  float64 `json:"distance"`
	Speed     float64 `json:"speed"`
	StartTime int64   `json:"start_time"`
}

func migrateExercise(dir string) {
	exDir := filepath.Join(dir, "com.samsung.shealth.exercise")
	if _, err := os.Stat(exDir); os.IsNotExist(err) {
		log.Println("Exercise: directory not found")
		return
	}

	// Group live_data files by UUID
	uuidFiles := map[string]string{}
	filepath.WalkDir(exDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		name := d.Name()
		if strings.Contains(name, "live_data") && strings.HasSuffix(name, ".json") {
			uuid := strings.Split(name, ".")[0]
			// Prefer external live_data over internal
			if _, exists := uuidFiles[uuid]; !exists || strings.Contains(name, "com.samsung.health") {
				uuidFiles[uuid] = path
			}
		}
		return nil
	})

	var exercises []ExerciseSession
	count := 0
	for _, path := range uuidFiles {
		if limit > 0 && count >= limit {
			break
		}

		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var entries []shExerciseLive
		if json.Unmarshal(data, &entries) != nil || len(entries) == 0 {
			continue
		}

		// Find min/max time, sum calories and distance
		var minT, maxT int64
		var totalCal, totalDist float64
		for _, e := range entries {
			if e.StartTime == 0 {
				continue
			}
			if minT == 0 || e.StartTime < minT {
				minT = e.StartTime
			}
			if e.StartTime > maxT {
				maxT = e.StartTime
			}
			totalCal += e.Calorie
			totalDist += e.Distance
		}

		if minT == 0 {
			continue
		}

		start := time.UnixMilli(minT)
		end := time.UnixMilli(maxT)
		dur := int(end.Sub(start).Minutes())
		if dur <= 0 {
			dur = 1
		}

		ex := ExerciseSession{
			ExerciseType: "unknown", // Samsung doesn't store type in live_data
			StartTime:    start,
			EndTime:      end,
			DurationM:    dur,
		}
		if totalCal > 0 {
			ex.CaloriesKcal = &totalCal
		}
		if totalDist > 0 {
			ex.DistanceM = &totalDist
		}
		exercises = append(exercises, ex)
		count++
	}

	log.Printf("Exercise: %d UUIDs → %d sessions", len(uuidFiles), len(exercises))
	sendExercises(exercises)
}

// ── Sleep ───────────────────────────────────────────────────
// sleep_data has {status, start_time, binning_period} entries
// status: 0=awake, 40=light, 90=deep?, 100=deep/REM?
// Group consecutive entries into sessions (gap > 2h = new session)

type shSleepEntry struct {
	Status        int   `json:"status"`
	StartTime     int64 `json:"start_time"`
	BinningPeriod int   `json:"binning_period"` // minutes
}

func migrateSleep(dir string) {
	records := collectJSON[shSleepEntry](dir, "com.samsung.shealth.sleep_data")
	if len(records) == 0 {
		log.Println("Sleep: no records found")
		return
	}

	// Samsung sleep status codes → stage names
	statusToStage := func(s int) string {
		switch {
		case s == 0:
			return "awake"
		case s <= 50:
			return "light"
		case s <= 90:
			return "deep"
		default:
			return "rem"
		}
	}

	// Sort by start_time and group into sessions
	// Simple approach: consecutive entries within 2h gap = same session
	var sessions []SleepSession
	var current *SleepSession
	var stages []SleepStage

	for _, r := range records {
		if r.StartTime == 0 {
			continue
		}
		t := time.UnixMilli(r.StartTime)
		binMin := r.BinningPeriod
		if binMin <= 0 {
			binMin = 10
		}
		endT := t.Add(time.Duration(binMin) * time.Minute)

		if current != nil && t.Sub(current.EndTime) > 2*time.Hour {
			// Finalize previous session
			current.Stages = stages
			current.DurationM = int(current.EndTime.Sub(current.StartTime).Minutes())
			if current.DurationM > 0 {
				sessions = append(sessions, *current)
			}
			current = nil
			stages = nil
		}

		if current == nil {
			current = &SleepSession{StartTime: t, EndTime: endT}
		}
		if endT.After(current.EndTime) {
			current.EndTime = endT
		}

		stages = append(stages, SleepStage{
			Stage: statusToStage(r.Status), StartTime: t, EndTime: endT, DurationM: binMin,
		})
	}

	if current != nil {
		current.Stages = stages
		current.DurationM = int(current.EndTime.Sub(current.StartTime).Minutes())
		if current.DurationM > 0 {
			sessions = append(sessions, *current)
		}
	}

	log.Printf("Sleep: %d entries → %d sessions", len(records), len(sessions))

	// Send in batches of 20
	for i := 0; i < len(sessions); i += 20 {
		end := i + 20
		if end > len(sessions) {
			end = len(sessions)
		}
		payload := IngestPayload{Timestamp: time.Now(), Sleep: sessions[i:end]}
		sendPayload(payload)
		log.Printf("  Sleep: sent batch %d-%d", i, end-1)
	}
}

// ── Helpers ─────────────────────────────────────────────────

func collectJSON[T any](baseDir, subdir string) []T {
	targetDir := filepath.Join(baseDir, subdir)
	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		log.Printf("%s: directory not found", subdir)
		return nil
	}

	var all []T
	filepath.WalkDir(targetDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".json") {
			return nil
		}
		// Skip extra_data, source_info, achievement, sensing files
		name := d.Name()
		if strings.Contains(name, "extra_data") || strings.Contains(name, "source_info") ||
			strings.Contains(name, "achievement") || strings.Contains(name, "sensing") ||
			strings.Contains(name, "additional") || strings.Contains(name, "location") {
			return nil
		}

		if limit > 0 && len(all) >= limit {
			return filepath.SkipAll
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		var items []T
		if json.Unmarshal(data, &items) == nil {
			remaining := len(items)
			if limit > 0 {
				remaining = limit - len(all)
				if remaining <= 0 {
					return filepath.SkipAll
				}
				if remaining < len(items) {
					items = items[:remaining]
				}
			}
			all = append(all, items...)
			return nil
		}

		var item T
		if json.Unmarshal(data, &item) == nil {
			all = append(all, item)
		}
		return nil
	})

	return all
}

func sendMetrics(metrics []Metric) {
	for i := 0; i < len(metrics); i += batchSize {
		end := i + batchSize
		if end > len(metrics) {
			end = len(metrics)
		}
		payload := IngestPayload{Timestamp: time.Now(), Metrics: metrics[i:end]}
		sendPayload(payload)
		log.Printf("  Sent metrics batch %d-%d (%d)", i, end-1, end-i)
	}
}

func sendExercises(exercises []ExerciseSession) {
	for i := 0; i < len(exercises); i += 50 {
		end := i + 50
		if end > len(exercises) {
			end = len(exercises)
		}
		payload := IngestPayload{Timestamp: time.Now(), Exercises: exercises[i:end]}
		sendPayload(payload)
		log.Printf("  Sent exercises batch %d-%d (%d)", i, end-1, end-i)
	}
}

func sendPayload(payload IngestPayload) {
	body, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", apiURL+"/api/v1/ingest", bytes.NewReader(body))
	if err != nil {
		log.Printf("  ERROR: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("  ERROR: %v", err)
		return
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	if resp.StatusCode != 200 {
		log.Printf("  ERROR HTTP %d: %v", resp.StatusCode, result)
	}
}
