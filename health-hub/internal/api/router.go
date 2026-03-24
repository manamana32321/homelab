package api

import (
	"net/http"
)

func NewRouter(repo DataRepository, apiToken string) http.Handler {
	mux := http.NewServeMux()

	h := &handler{repo: repo}

	mux.HandleFunc("GET /healthz", h.healthz)
	mux.HandleFunc("POST /api/v1/ingest", h.ingest)
	mux.HandleFunc("GET /api/v1/metrics", h.queryMetrics)
	mux.HandleFunc("GET /api/v1/sleep", h.querySleep)
	mux.HandleFunc("GET /api/v1/exercises", h.queryExercises)
	mux.HandleFunc("GET /api/v1/nutrition", h.queryNutrition)
	mux.HandleFunc("GET /api/v1/body", h.queryBody)
	mux.HandleFunc("GET /api/v1/summary", h.summary)
	mux.HandleFunc("POST /api/v1/meals", h.addMeal)
	mux.HandleFunc("POST /api/v1/hc-webhook", h.hcWebhook)
	mux.HandleFunc("DELETE /api/v1/admin/purge", h.purgeData)

	var root http.Handler = mux
	root = loggingMiddleware(root)
	if apiToken != "" {
		root = authMiddleware(apiToken, root)
	}

	return root
}
