package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/manamana32321/homelab/health-hub/internal/db"
	"github.com/manamana32321/homelab/health-hub/internal/model"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dbHost := envOr("DB_HOST", "localhost")
		dbPort := envOr("DB_PORT", "5432")
		dbUser := envOr("DB_USER", "healthhub")
		dbPass := envOr("DB_PASSWORD", "healthhub")
		dbName := envOr("DB_NAME", "healthhub")
		dsn = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", dbUser, dbPass, dbHost, dbPort, dbName)
	}

	listenAddr := envOr("LISTEN_ADDR", ":8081")
	baseURL := envOr("BASE_URL", "http://localhost:8081")

	pool, err := db.Connect(context.Background(), dsn)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	repo := db.NewRepository(pool)

	s := server.NewMCPServer(
		"health-hub",
		"1.0.0",
		server.WithToolCapabilities(true),
	)

	registerTools(s, repo)

	sseServer := server.NewSSEServer(s,
		server.WithBaseURL(baseURL),
	)

	slog.Info("MCP SSE server starting", "addr", listenAddr, "base_url", baseURL)
	if err := sseServer.Start(listenAddr); err != nil {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}
}

func registerTools(s *server.MCPServer, repo *db.Repository) {
	// get_daily_summary
	s.AddTool(
		mcp.NewTool("get_daily_summary",
			mcp.WithDescription("오늘 또는 특정 날짜의 건강 요약 (걸음수, 평균 심박수, 수면, 운동, 칼로리, SpO2, 체중)"),
			mcp.WithString("date",
				mcp.Description("조회할 날짜 (YYYY-MM-DD). 생략하면 오늘."),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			dateStr := req.GetString("date", "")
			var date time.Time
			if dateStr == "" {
				date = time.Now()
			} else {
				var err error
				date, err = time.Parse("2006-01-02", dateStr)
				if err != nil {
					return mcp.NewToolResultError("날짜 형식이 올바르지 않습니다. YYYY-MM-DD 형식으로 입력하세요."), nil
				}
			}

			summary, err := repo.GetDailySummary(ctx, date)
			if err != nil {
				return mcp.NewToolResultError("조회 실패: " + err.Error()), nil
			}

			b, _ := json.MarshalIndent(summary, "", "  ")
			return mcp.NewToolResultText(string(b)), nil
		},
	)

	// get_steps
	s.AddTool(
		mcp.NewTool("get_steps",
			mcp.WithDescription("걸음수 조회. 시간대별 집계를 반환합니다."),
			mcp.WithString("from", mcp.Description("시작일 (YYYY-MM-DD 또는 RFC3339). 생략하면 7일 전.")),
			mcp.WithString("to", mcp.Description("종료일. 생략하면 현재.")),
			mcp.WithString("interval", mcp.Description("집계 간격: 1h, 1d 등. 기본 1d.")),
		),
		makeMetricHandler(repo, "steps"),
	)

	// get_heart_rate
	s.AddTool(
		mcp.NewTool("get_heart_rate",
			mcp.WithDescription("심박수 조회. 평균/최소/최대를 시간대별로 반환합니다."),
			mcp.WithString("from", mcp.Description("시작일 (YYYY-MM-DD 또는 RFC3339). 생략하면 7일 전.")),
			mcp.WithString("to", mcp.Description("종료일. 생략하면 현재.")),
			mcp.WithString("interval", mcp.Description("집계 간격: 5m, 1h, 1d 등. 기본 1h.")),
		),
		makeMetricHandler(repo, "heart_rate"),
	)

	// get_sleep
	s.AddTool(
		mcp.NewTool("get_sleep",
			mcp.WithDescription("수면 기록 조회. 세션별 시작/종료/시간/단계 정보를 반환합니다."),
			mcp.WithString("from", mcp.Description("시작일 (YYYY-MM-DD). 생략하면 7일 전.")),
			mcp.WithString("to", mcp.Description("종료일. 생략하면 현재.")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			from, to := parseRange(req)
			sessions, err := repo.QuerySleepSessions(ctx, model.TimeRangeQuery{From: from, To: to})
			if err != nil {
				return mcp.NewToolResultError("조회 실패: " + err.Error()), nil
			}
			if sessions == nil {
				sessions = []model.SleepSession{}
			}
			b, _ := json.MarshalIndent(sessions, "", "  ")
			return mcp.NewToolResultText(string(b)), nil
		},
	)

	// get_exercises
	s.AddTool(
		mcp.NewTool("get_exercises",
			mcp.WithDescription("운동 기록 조회. 종류, 시간, 칼로리, 거리 등을 반환합니다."),
			mcp.WithString("from", mcp.Description("시작일 (YYYY-MM-DD). 생략하면 7일 전.")),
			mcp.WithString("to", mcp.Description("종료일. 생략하면 현재.")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			from, to := parseRange(req)
			sessions, err := repo.QueryExerciseSessions(ctx, model.TimeRangeQuery{From: from, To: to})
			if err != nil {
				return mcp.NewToolResultError("조회 실패: " + err.Error()), nil
			}
			if sessions == nil {
				sessions = []model.ExerciseSession{}
			}
			b, _ := json.MarshalIndent(sessions, "", "  ")
			return mcp.NewToolResultText(string(b)), nil
		},
	)

	// get_weight
	s.AddTool(
		mcp.NewTool("get_weight",
			mcp.WithDescription("체중 및 체성분 기록 조회."),
			mcp.WithString("from", mcp.Description("시작일 (YYYY-MM-DD). 생략하면 30일 전.")),
			mcp.WithString("to", mcp.Description("종료일. 생략하면 현재.")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			fromStr := req.GetString("from", "")
			toStr := req.GetString("to", "")
			from, to := defaults(fromStr, toStr, 30)
			measurements, err := repo.QueryBodyMeasurements(ctx, model.TimeRangeQuery{From: from, To: to})
			if err != nil {
				return mcp.NewToolResultError("조회 실패: " + err.Error()), nil
			}
			if measurements == nil {
				measurements = []model.BodyMeasurement{}
			}
			b, _ := json.MarshalIndent(measurements, "", "  ")
			return mcp.NewToolResultText(string(b)), nil
		},
	)

	// get_spo2
	s.AddTool(
		mcp.NewTool("get_spo2",
			mcp.WithDescription("산소포화도(SpO2) 조회."),
			mcp.WithString("from", mcp.Description("시작일. 생략하면 7일 전.")),
			mcp.WithString("to", mcp.Description("종료일. 생략하면 현재.")),
			mcp.WithString("interval", mcp.Description("집계 간격. 기본 1d.")),
		),
		makeMetricHandler(repo, "spo2"),
	)

	// get_calories
	s.AddTool(
		mcp.NewTool("get_calories",
			mcp.WithDescription("소모 칼로리 조회."),
			mcp.WithString("from", mcp.Description("시작일. 생략하면 7일 전.")),
			mcp.WithString("to", mcp.Description("종료일. 생략하면 현재.")),
			mcp.WithString("interval", mcp.Description("집계 간격. 기본 1d.")),
		),
		makeMetricHandler(repo, "calories"),
	)

	// get_distance
	s.AddTool(
		mcp.NewTool("get_distance",
			mcp.WithDescription("이동 거리 조회 (미터 단위)."),
			mcp.WithString("from", mcp.Description("시작일. 생략하면 7일 전.")),
			mcp.WithString("to", mcp.Description("종료일. 생략하면 현재.")),
			mcp.WithString("interval", mcp.Description("집계 간격. 기본 1d.")),
		),
		makeMetricHandler(repo, "distance"),
	)

	// add_meal — 식단 기록 추가
	s.AddTool(
		mcp.NewTool("add_meal",
			mcp.WithDescription("식단/식사를 기록합니다. 사용자가 먹은 음식을 Health Hub에 저장합니다."),
			mcp.WithString("name", mcp.Description("음식 이름 (예: '고등어구이, 계란프라이, 샐러드'). 필수."), mcp.Required()),
			mcp.WithString("meal_type", mcp.Description("식사 종류: breakfast, lunch, dinner, snack. 생략 가능.")),
			mcp.WithNumber("calories", mcp.Description("칼로리 (kcal). 모르면 생략.")),
			mcp.WithNumber("protein_g", mcp.Description("단백질 (g). 모르면 생략.")),
			mcp.WithNumber("fat_g", mcp.Description("지방 (g). 모르면 생략.")),
			mcp.WithNumber("carbs_g", mcp.Description("탄수화물 (g). 모르면 생략.")),
			mcp.WithString("notes", mcp.Description("메모 (예: '클린식', '설밀나튀 0'). 생략 가능.")),
			mcp.WithString("time", mcp.Description("식사 시간 (RFC3339 또는 HH:MM). 생략하면 현재 시각.")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			name := req.GetString("name", "")
			if name == "" {
				return mcp.NewToolResultError("음식 이름(name)은 필수입니다."), nil
			}

			meal := model.NutritionRecord{
				Name: &name,
				Time: time.Now(),
			}

			if mt := req.GetString("meal_type", ""); mt != "" {
				meal.MealType = &mt
			}
			if notes := req.GetString("notes", ""); notes != "" {
				meal.Notes = &notes
			}
			if timeStr := req.GetString("time", ""); timeStr != "" {
				if t, err := time.Parse(time.RFC3339, timeStr); err == nil {
					meal.Time = t
				} else if t, err := time.Parse("15:04", timeStr); err == nil {
					now := time.Now()
					meal.Time = time.Date(now.Year(), now.Month(), now.Day(), t.Hour(), t.Minute(), 0, 0, now.Location())
				}
			}

			args := req.GetArguments()
			if v, ok := args["calories"].(float64); ok {
				meal.Calories = &v
			}
			if v, ok := args["protein_g"].(float64); ok {
				meal.ProteinG = &v
			}
			if v, ok := args["fat_g"].(float64); ok {
				meal.FatG = &v
			}
			if v, ok := args["carbs_g"].(float64); ok {
				meal.CarbsG = &v
			}

			id, err := repo.InsertNutritionRecord(ctx, meal)
			if err != nil {
				return mcp.NewToolResultError("저장 실패: " + err.Error()), nil
			}

			return mcp.NewToolResultText(fmt.Sprintf("식단 기록 완료 (ID: %d): %s", id, name)), nil
		},
	)

	// get_meals — 식단 기록 조회
	s.AddTool(
		mcp.NewTool("get_meals",
			mcp.WithDescription("식단/식사 기록을 조회합니다. 음식명, 영양소, 메모 등을 반환합니다."),
			mcp.WithString("from", mcp.Description("시작일 (YYYY-MM-DD). 생략하면 7일 전.")),
			mcp.WithString("to", mcp.Description("종료일. 생략하면 현재.")),
			mcp.WithString("meal_type", mcp.Description("필터: breakfast, lunch, dinner, snack. 생략하면 전체.")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			from, to := parseRange(req)
			mealType := req.GetString("meal_type", "")
			records, err := repo.QueryNutritionByType(ctx, model.TimeRangeQuery{From: from, To: to}, mealType)
			if err != nil {
				return mcp.NewToolResultError("조회 실패: " + err.Error()), nil
			}
			if records == nil {
				records = []model.NutritionRecord{}
			}
			b, _ := json.MarshalIndent(records, "", "  ")
			return mcp.NewToolResultText(string(b)), nil
		},
	)

	// update_meal — 식단 기록 수정
	s.AddTool(
		mcp.NewTool("update_meal",
			mcp.WithDescription("기존 식단 기록을 수정합니다. ID로 특정 기록을 찾아 수정합니다."),
			mcp.WithNumber("id", mcp.Description("수정할 식단 기록 ID. 필수."), mcp.Required()),
			mcp.WithString("name", mcp.Description("음식 이름.")),
			mcp.WithString("meal_type", mcp.Description("식사 종류.")),
			mcp.WithNumber("calories", mcp.Description("칼로리 (kcal).")),
			mcp.WithNumber("protein_g", mcp.Description("단백질 (g).")),
			mcp.WithNumber("fat_g", mcp.Description("지방 (g).")),
			mcp.WithNumber("carbs_g", mcp.Description("탄수화물 (g).")),
			mcp.WithString("notes", mcp.Description("메모.")),
			mcp.WithString("time", mcp.Description("식사 시간 (RFC3339).")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.GetArguments()
			idFloat, ok := args["id"].(float64)
			if !ok {
				return mcp.NewToolResultError("ID는 필수입니다."), nil
			}
			id := int64(idFloat)

			existing, err := repo.GetNutritionRecord(ctx, id)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("기록을 찾을 수 없습니다 (ID: %d)", id)), nil
			}

			if v := req.GetString("name", ""); v != "" {
				existing.Name = &v
			}
			if v := req.GetString("meal_type", ""); v != "" {
				existing.MealType = &v
			}
			if v := req.GetString("notes", ""); v != "" {
				existing.Notes = &v
			}
			if v := req.GetString("time", ""); v != "" {
				if t, err := time.Parse(time.RFC3339, v); err == nil {
					existing.Time = t
				}
			}
			if v, ok := args["calories"].(float64); ok {
				existing.Calories = &v
			}
			if v, ok := args["protein_g"].(float64); ok {
				existing.ProteinG = &v
			}
			if v, ok := args["fat_g"].(float64); ok {
				existing.FatG = &v
			}
			if v, ok := args["carbs_g"].(float64); ok {
				existing.CarbsG = &v
			}

			if err := repo.UpdateNutritionRecord(ctx, *existing); err != nil {
				return mcp.NewToolResultError("수정 실패: " + err.Error()), nil
			}

			b, _ := json.MarshalIndent(existing, "", "  ")
			return mcp.NewToolResultText(fmt.Sprintf("식단 기록 수정 완료 (ID: %d)\n%s", id, string(b))), nil
		},
	)

	// delete_meal — 식단 기록 삭제
	s.AddTool(
		mcp.NewTool("delete_meal",
			mcp.WithDescription("식단 기록을 삭제합니다."),
			mcp.WithNumber("id", mcp.Description("삭제할 식단 기록 ID. 필수."), mcp.Required()),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.GetArguments()
			idFloat, ok := args["id"].(float64)
			if !ok {
				return mcp.NewToolResultError("ID는 필수입니다."), nil
			}
			id := int64(idFloat)

			if err := repo.DeleteNutritionRecord(ctx, id); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("삭제 실패: %v", err)), nil
			}

			return mcp.NewToolResultText(fmt.Sprintf("식단 기록 삭제 완료 (ID: %d)", id)), nil
		},
	)

	// add_weight — 체중 수동 입력
	s.AddTool(
		mcp.NewTool("add_weight",
			mcp.WithDescription("체중을 수동으로 기록합니다."),
			mcp.WithNumber("weight_kg", mcp.Description("체중 (kg). 필수."), mcp.Required()),
			mcp.WithNumber("body_fat_pct", mcp.Description("체지방률 (%). 모르면 생략.")),
			mcp.WithString("time", mcp.Description("측정 시간 (RFC3339). 생략하면 현재 시각.")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.GetArguments()
			wt, ok := args["weight_kg"].(float64)
			if !ok || wt <= 0 {
				return mcp.NewToolResultError("체중(weight_kg)은 필수입니다."), nil
			}

			bm := model.BodyMeasurement{
				Time:     time.Now(),
				WeightKg: &wt,
			}

			if v, ok := args["body_fat_pct"].(float64); ok {
				bm.BodyFatPct = &v
			}
			if v := req.GetString("time", ""); v != "" {
				if t, err := time.Parse(time.RFC3339, v); err == nil {
					bm.Time = t
				}
			}

			if err := repo.InsertBodyMeasurement(ctx, bm); err != nil {
				return mcp.NewToolResultError("저장 실패: " + err.Error()), nil
			}

			return mcp.NewToolResultText(fmt.Sprintf("체중 기록 완료: %.1fkg", wt)), nil
		},
	)

	// add_note — 건강 메모/특이사항 기록
	s.AddTool(
		mcp.NewTool("add_note",
			mcp.WithDescription("건강 메모/특이사항을 기록합니다. 증상, 컨디션, 음주 등."),
			mcp.WithString("text", mcp.Description("내용. 필수."), mcp.Required()),
			mcp.WithString("category", mcp.Description("카테고리: symptom, condition, memo. 기본 memo.")),
			mcp.WithString("time", mcp.Description("시간 (RFC3339). 생략하면 현재 시각.")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			text := req.GetString("text", "")
			if text == "" {
				return mcp.NewToolResultError("내용(text)은 필수입니다."), nil
			}

			note := model.HealthNote{
				Text:     text,
				Category: req.GetString("category", "memo"),
				Time:     time.Now(),
			}

			if v := req.GetString("time", ""); v != "" {
				if t, err := time.Parse(time.RFC3339, v); err == nil {
					note.Time = t
				}
			}

			id, err := repo.InsertHealthNote(ctx, note)
			if err != nil {
				return mcp.NewToolResultError("저장 실패: " + err.Error()), nil
			}

			return mcp.NewToolResultText(fmt.Sprintf("건강 메모 기록 완료 (ID: %d, %s): %s", id, note.Category, text)), nil
		},
	)

	// get_notes — 건강 메모 조회
	s.AddTool(
		mcp.NewTool("get_notes",
			mcp.WithDescription("건강 메모/특이사항을 조회합니다."),
			mcp.WithString("from", mcp.Description("시작일 (YYYY-MM-DD). 생략하면 7일 전.")),
			mcp.WithString("to", mcp.Description("종료일. 생략하면 현재.")),
			mcp.WithString("category", mcp.Description("필터: symptom, condition, memo. 생략하면 전체.")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			from, to := parseRange(req)
			category := req.GetString("category", "")
			notes, err := repo.QueryHealthNotes(ctx, model.TimeRangeQuery{From: from, To: to}, category)
			if err != nil {
				return mcp.NewToolResultError("조회 실패: " + err.Error()), nil
			}
			if notes == nil {
				notes = []model.HealthNote{}
			}
			b, _ := json.MarshalIndent(notes, "", "  ")
			return mcp.NewToolResultText(string(b)), nil
		},
	)
}

func makeMetricHandler(repo *db.Repository, metricType string) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		from, to := parseRange(req)
		interval := req.GetString("interval", "1d")

		results, err := repo.QueryMetrics(ctx, model.MetricsQuery{
			Type:     metricType,
			From:     from,
			To:       to,
			Interval: interval,
		})
		if err != nil {
			return mcp.NewToolResultError("조회 실패: " + err.Error()), nil
		}
		if results == nil {
			results = []model.AggregatedMetric{}
		}
		b, _ := json.MarshalIndent(results, "", "  ")
		return mcp.NewToolResultText(string(b)), nil
	}
}

func parseRange(req mcp.CallToolRequest) (time.Time, time.Time) {
	fromStr := req.GetString("from", "")
	toStr := req.GetString("to", "")
	return defaults(fromStr, toStr, 7)
}

func defaults(fromStr, toStr string, defaultDays int) (time.Time, time.Time) {
	var from, to time.Time
	now := time.Now()

	if fromStr == "" {
		from = now.AddDate(0, 0, -defaultDays)
	} else {
		from, _ = time.Parse("2006-01-02", fromStr)
		if from.IsZero() {
			from, _ = time.Parse(time.RFC3339, fromStr)
		}
		if from.IsZero() {
			from = now.AddDate(0, 0, -defaultDays)
		}
	}

	if toStr == "" {
		to = now
	} else {
		to, _ = time.Parse("2006-01-02", toStr)
		if to.IsZero() {
			to, _ = time.Parse(time.RFC3339, toStr)
		}
		if to.IsZero() {
			to = now
		} else if to.Hour() == 0 && to.Minute() == 0 {
			to = to.Add(24 * time.Hour)
		}
	}

	return from, to
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
