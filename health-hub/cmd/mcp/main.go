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
