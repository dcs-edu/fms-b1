package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"time"
)

// GET /healthz
// For nginx, Docker or an uptime checker: 200 when the server can reach the database, 503 when it can't.
// The page on / only proves the database was up when the server started.
func (srv *Pool) HealthHandler(w http.ResponseWriter, r *http.Request) {
	// a checker that waits forever is no use, so the ping gets 2 seconds
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if err := srv.Repo.DB.PingContext(ctx); err != nil {
		slog.ErrorContext(ctx, "health check: database unreachable", "error", err)
		http.Error(w, "503 database unreachable", http.StatusServiceUnavailable)
		return
	}
	writeJSON(w, r, http.StatusOK, map[string]string{"status": "ok"})
}
