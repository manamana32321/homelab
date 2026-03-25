package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
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
	ExerciseType string    `json:"exercise_type"`
	StartTime    time.Time `json:"start_time"`
	EndTime      time.Time `json:"end_time"`
	DurationM    int       `json:"duration_m"`
	CaloriesKcal *float64  `json:"calories_kcal,omitempty"`
	DistanceM    *float64  `json:"distance_m,omitempty"`
}

type BodyMeasurement struct {
	Time       time.Time `json:"time"`
	WeightKg   *float64  `json:"weight_kg,omitempty"`
	BodyFatPct *float64  `json:"body_fat_pct,omitempty"`
}

type IngestPayload struct {
	Timestamp time.Time           `json:"timestamp"`
	Metrics   []Metric            `json:"metrics,omitempty"`
	Sleep     []SleepSession      `json:"sleep_sessions,omitempty"`
	Exercises []ExerciseSession   `json:"exercises,omitempty"`
	Body      []BodyMeasurement   `json:"body,omitempty"`
}

var (
	apiURL string
	token  string
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: migrate-csv <samsung-health-csv-dir> <api-url> [api-token]\n")
		os.Exit(1)
	}

	dir := os.Args[1]
	apiURL = strings.TrimRight(os.Args[2], "/")
	if len(os.Args) > 3 {
		token = os.Args[3]
	}

	log.Printf("Samsung Health CSV migration → %s", apiURL)

	migrateWeight(dir)
	migrateSleepStages(dir)
	migrateHeartRate(dir)
	migrateSteps(dir)
	migrateExercise(dir)

	log.Println("Done!")
}

func findCSVFile(dir, prefix string) string {
	matches, _ := filepath.Glob(filepath.Join(dir, prefix+"*.csv"))
	if len(matches) > 0 {
		return matches[0]
	}
	return ""
}

func readCSV(path string) ([]map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Remove BOM
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))

	// Normalize line endings (CR LF → LF)
	data = bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))
	data = bytes.ReplaceAll(data, []byte("\r"), []byte("\n"))

	// Skip first line if it's metadata (com.samsung.health...)
	lines := strings.SplitN(string(data), "\n", 2)
	if len(lines) < 2 {
		return nil, fmt.Errorf("file too short")
	}
	if strings.HasPrefix(lines[0], "com.samsung") {
		data = []byte(lines[1])
	}

	r := csv.NewReader(bytes.NewReader(data))
	r.LazyQuotes = true
	r.TrimLeadingSpace = true
	r.FieldsPerRecord = -1 // allow variable field count (Samsung CSV has trailing commas)

	headers, err := r.Read()
	if err != nil {
		return nil, err
	}

	var records []map[string]string
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}
		record := make(map[string]string)
		for i, h := range headers {
			if i < len(row) {
				record[strings.TrimSpace(h)] = strings.TrimSpace(row[i])
			}
		}
		records = append(records, record)
	}
	return records, nil
}

func parseSamsungTime(s, offset string) time.Time {
	if s == "" {
		return time.Time{}
	}

	// Parse timezone offset like "UTC+0900" → "+09:00"
	loc := time.UTC
	if strings.HasPrefix(offset, "UTC") {
		offsetStr := strings.TrimPrefix(offset, "UTC")
		if len(offsetStr) == 5 { // +0900
			hours, _ := strconv.Atoi(offsetStr[:3])
			mins, _ := strconv.Atoi(offsetStr[3:5])
			loc = time.FixedZone("custom", hours*3600+mins*60)
		}
	}

	formats := []string{
		"2006-01-02 15:04:05.000",
		"2006-01-02 15:04:05",
	}

	for _, f := range formats {
		t, err := time.ParseInLocation(f, s, loc)
		if err == nil {
			return t.UTC()
		}
	}
	return time.Time{}
}

// ── Weight ──────────────────────────────────────────────

func migrateWeight(dir string) {
	path := findCSVFile(dir, "com.samsung.health.weight.")
	if path == "" {
		log.Println("Weight: no CSV found")
		return
	}

	records, err := readCSV(path)
	if err != nil {
		log.Printf("Weight: error reading: %v", err)
		return
	}

	log.Printf("Weight: %d records", len(records))

	var body []BodyMeasurement
	for _, r := range records {
		t := parseSamsungTime(r["start_time"], r["time_offset"])
		if t.IsZero() {
			continue
		}

		bm := BodyMeasurement{Time: t}
		hasData := false

		if w, err := strconv.ParseFloat(r["weight"], 64); err == nil && w > 0 {
			bm.WeightKg = &w
			hasData = true
		}
		if f, err := strconv.ParseFloat(r["body_fat"], 64); err == nil && f > 0 {
			bm.BodyFatPct = &f
			hasData = true
		}

		if hasData {
			body = append(body, bm)
		}
	}

	log.Printf("Weight: %d valid records", len(body))

	// Send in batches of 100
	for i := 0; i < len(body); i += 100 {
		end := i + 100
		if end > len(body) {
			end = len(body)
		}
		sendPayload(IngestPayload{Timestamp: time.Now(), Body: body[i:end]})
		log.Printf("  Weight: sent batch %d-%d", i, end-1)
	}
}

// ── Sleep Stages ────────────────────────────────────────

func migrateSleepStages(dir string) {
	path := findCSVFile(dir, "com.samsung.health.sleep_stage.")
	if path == "" {
		log.Println("Sleep: no CSV found")
		return
	}

	records, err := readCSV(path)
	if err != nil {
		log.Printf("Sleep: error reading: %v", err)
		return
	}

	log.Printf("Sleep stages: %d records", len(records))

	// Group by sleep_id
	sleepMap := map[string][]struct {
		Stage     int
		StartTime time.Time
		EndTime   time.Time
	}{}

	for _, r := range records {
		sleepID := r["sleep_id"]
		if sleepID == "" {
			continue
		}

		startT := parseSamsungTime(r["start_time"], r["time_offset"])
		endT := parseSamsungTime(r["end_time"], r["time_offset"])
		if startT.IsZero() || endT.IsZero() {
			continue
		}

		stageCode, _ := strconv.Atoi(r["stage"])

		sleepMap[sleepID] = append(sleepMap[sleepID], struct {
			Stage     int
			StartTime time.Time
			EndTime   time.Time
		}{stageCode, startT, endT})
	}

	log.Printf("Sleep: %d unique sessions", len(sleepMap))

	// Samsung sleep stage codes:
	// 40001=Awake, 40002=Light, 40003=Deep, 40004=REM
	stageToName := func(code int) string {
		switch code {
		case 40001:
			return "awake"
		case 40002:
			return "light"
		case 40003:
			return "deep"
		case 40004:
			return "rem"
		default:
			return fmt.Sprintf("%d", code)
		}
	}

	var sessions []SleepSession
	for _, entries := range sleepMap {
		if len(entries) == 0 {
			continue
		}

		// Sort by start time
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].StartTime.Before(entries[j].StartTime)
		})

		sessionStart := entries[0].StartTime
		sessionEnd := entries[len(entries)-1].EndTime

		var stages []SleepStage
		for _, e := range entries {
			dur := int(e.EndTime.Sub(e.StartTime).Minutes())
			if dur <= 0 {
				dur = 1
			}
			stages = append(stages, SleepStage{
				Stage:     stageToName(e.Stage),
				StartTime: e.StartTime,
				EndTime:   e.EndTime,
				DurationM: dur,
			})
		}

		totalDur := int(sessionEnd.Sub(sessionStart).Minutes())
		if totalDur <= 0 {
			continue
		}

		sessions = append(sessions, SleepSession{
			StartTime: sessionStart,
			EndTime:   sessionEnd,
			DurationM: totalDur,
			Stages:    stages,
		})
	}

	log.Printf("Sleep: %d sessions built", len(sessions))

	// Send in batches of 20
	for i := 0; i < len(sessions); i += 20 {
		end := i + 20
		if end > len(sessions) {
			end = len(sessions)
		}
		sendPayload(IngestPayload{Timestamp: time.Now(), Sleep: sessions[i:end]})
		log.Printf("  Sleep: sent batch %d-%d", i, end-1)
	}
}

// ── Heart Rate ──────────────────────────────────────────

func migrateHeartRate(dir string) {
	path := findCSVFile(dir, "com.samsung.shealth.tracker.heart_rate.")
	if path == "" {
		log.Println("HeartRate: no CSV found")
		return
	}

	records, err := readCSV(path)
	if err != nil {
		log.Printf("HeartRate: error reading: %v", err)
		return
	}

	log.Printf("HeartRate: %d records", len(records))

	var metrics []Metric
	for _, r := range records {
		timeStr := r["com.samsung.health.heart_rate.start_time"]
		hrStr := r["com.samsung.health.heart_rate.heart_rate"]
		offset := r["com.samsung.health.heart_rate.time_offset"]

		t := parseSamsungTime(timeStr, offset)
		if t.IsZero() {
			continue
		}
		hr, err := strconv.ParseFloat(hrStr, 64)
		if err != nil || hr <= 0 || hr > 300 {
			continue
		}

		metrics = append(metrics, Metric{
			Time: t, MetricType: "heart_rate", Value: hr, Unit: "bpm", Source: "samsung_export",
		})
	}

	log.Printf("HeartRate: %d valid", len(metrics))
	sendMetricsBatch(metrics)
}

// ── Steps ───────────────────────────────────────────────

func migrateSteps(dir string) {
	path := findCSVFile(dir, "com.samsung.shealth.step_daily_trend.")
	if path == "" {
		log.Println("Steps: no CSV found")
		return
	}

	records, err := readCSV(path)
	if err != nil {
		log.Printf("Steps: error reading: %v", err)
		return
	}

	log.Printf("Steps: %d records", len(records))

	var metrics []Metric
	for _, r := range records {
		dayTimeStr := r["day_time"]
		countStr := r["count"]
		distStr := r["distance"]
		calStr := r["calorie"]

		// day_time is Unix millis
		dayTimeMs, err := strconv.ParseInt(dayTimeStr, 10, 64)
		if err != nil || dayTimeMs == 0 {
			continue
		}
		t := time.UnixMilli(dayTimeMs)

		count, err := strconv.ParseFloat(countStr, 64)
		if err != nil || count <= 0 {
			continue
		}

		metrics = append(metrics, Metric{
			Time: t, MetricType: "steps", Value: count, Unit: "count", Source: "samsung_export",
		})

		if dist, err := strconv.ParseFloat(distStr, 64); err == nil && dist > 0 {
			metrics = append(metrics, Metric{
				Time: t, MetricType: "distance", Value: dist, Unit: "m", Source: "samsung_export",
			})
		}
		if cal, err := strconv.ParseFloat(calStr, 64); err == nil && cal > 0 {
			metrics = append(metrics, Metric{
				Time: t, MetricType: "calories", Value: cal, Unit: "kcal", Source: "samsung_export",
			})
		}
	}

	log.Printf("Steps: %d metric records", len(metrics))
	sendMetricsBatch(metrics)
}

// ── Exercise ────────────────────────────────────────────

func migrateExercise(dir string) {
	path := findCSVFile(dir, "com.samsung.shealth.exercise.")
	if path == "" {
		log.Println("Exercise: no CSV found")
		return
	}

	records, err := readCSV(path)
	if err != nil {
		log.Printf("Exercise: error reading: %v", err)
		return
	}

	log.Printf("Exercise: %d records", len(records))

	var exercises []ExerciseSession
	for _, r := range records {
		startStr := r["com.samsung.health.exercise.start_time"]
		endStr := r["com.samsung.health.exercise.end_time"]
		exType := r["com.samsung.health.exercise.exercise_type"]
		calStr := r["com.samsung.health.exercise.calorie"]
		distStr := r["com.samsung.health.exercise.distance"]
		durStr := r["com.samsung.health.exercise.duration"]
		offset := r["com.samsung.health.exercise.time_offset"]

		start := parseSamsungTime(startStr, offset)
		if start.IsZero() {
			continue
		}

		end := parseSamsungTime(endStr, offset)
		if end.IsZero() {
			// Try duration (milliseconds)
			if durMs, err := strconv.ParseInt(durStr, 10, 64); err == nil && durMs > 0 {
				end = start.Add(time.Duration(durMs) * time.Millisecond)
			} else {
				continue
			}
		}

		dur := int(end.Sub(start).Minutes())
		if dur <= 0 {
			dur = 1
		}

		ex := ExerciseSession{
			ExerciseType: exType,
			StartTime:    start,
			EndTime:      end,
			DurationM:    dur,
		}

		if cal, err := strconv.ParseFloat(calStr, 64); err == nil && cal > 0 {
			ex.CaloriesKcal = &cal
		}
		if dist, err := strconv.ParseFloat(distStr, 64); err == nil && dist > 0 {
			ex.DistanceM = &dist
		}

		exercises = append(exercises, ex)
	}

	log.Printf("Exercise: %d valid sessions", len(exercises))

	for i := 0; i < len(exercises); i += 50 {
		end := i + 50
		if end > len(exercises) {
			end = len(exercises)
		}
		sendPayload(IngestPayload{Timestamp: time.Now(), Exercises: exercises[i:end]})
		log.Printf("  Exercise: sent batch %d-%d", i, end-1)
	}
}

func sendMetricsBatch(metrics []Metric) {
	for i := 0; i < len(metrics); i += 500 {
		end := i + 500
		if end > len(metrics) {
			end = len(metrics)
		}
		sendPayload(IngestPayload{Timestamp: time.Now(), Metrics: metrics[i:end]})
		log.Printf("  Metrics: sent batch %d-%d", i, end-1)
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