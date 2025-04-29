package loadbalancer

import (
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"time"
)

func isBackendAlive(u *url.URL) bool {
	timeout := 2 * time.Second
	conn, err := net.DialTimeout("tcp", u.Host, timeout)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

func sendError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"code":    code,
		"message": message,
	}); err != nil {
		slog.Error("Failed to encode error response", "error", err, "code", code, "message", message)
	}
}

func GetAttemptsFromContext(r *http.Request) int {
	if attempts, ok := r.Context().Value(attemptsKey).(int); ok {
		slog.Debug("Retrieved attempts from context", "attempts", attempts)
		return attempts
	}
	return 0
}

func GetRetryFromContext(r *http.Request) int {
	if retry, ok := r.Context().Value(retryKey).(int); ok {
		slog.Debug("Retrieved retry count from context", "retry", retry)
		return retry
	}
	return 0
}