package loadbalancer

import (
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dorik33/cloud/internal/ratelimit"
)

type contextKey string

const (
	attemptsKey contextKey = "attempts"
	retryKey    contextKey = "retry"
)

type Backend struct {
	URL          *url.URL
	Alive        bool
	mux          sync.RWMutex
	ReverseProxy *httputil.ReverseProxy
}

func (b *Backend) SetAlive(alive bool) {
	b.mux.Lock()
	b.Alive = alive
	b.mux.Unlock()
}

func (b *Backend) IsAlive() bool {
	b.mux.RLock()
	alive := b.Alive
	b.mux.RUnlock()
	return alive
}

type ServerPool struct {
	backends       []*Backend
	currentBackend uint64
	rl             *ratelimit.RateLimiter
}

func NewServerPool(rl *ratelimit.RateLimiter) *ServerPool {
	return &ServerPool{
		rl: rl,
	}
}

func (s *ServerPool) AddBackend(url *url.URL) {
	rp := httputil.NewSingleHostReverseProxy(url)
	backend := Backend{
		URL:          url,
		Alive:        true,
		ReverseProxy: rp,
	}
	s.backends = append(s.backends, &backend)
}

func (s *ServerPool) NextIndex() int {
	return int(atomic.AddUint64(&s.currentBackend, uint64(1)) % uint64(len(s.backends)))
}

func (s *ServerPool) GetNextBackend() *Backend {
	next := s.NextIndex()
	l := len(s.backends) + next
	for i := next; i < l; i++ {
		idx := i % len(s.backends)
		if s.backends[idx].IsAlive() {
			if i != next {
				atomic.StoreUint64(&s.currentBackend, uint64(idx))
			}
			return s.backends[idx]
		}
	}
	return nil
}

func (s *ServerPool) HealthCheck() {
	for _, b := range s.backends {
		alive := isBackendAlive(b.URL)
		b.SetAlive(alive)
	}
}

func (s *ServerPool) StartHealthCheck(interval time.Duration) {
	go func() {
		t := time.NewTicker(interval)
		for {
			select {
			case <-t.C:
				slog.Debug("Starting health check...")
				s.HealthCheck()
				for _, b := range s.backends {
					status := "DOWN"
					if b.IsAlive() {
						status = "UP"
					}
					slog.Debug("Backend status", "url", b.URL, "status", status)
				}
				slog.Debug("Health check completed")
			}
		}
	}()
}

func (s *ServerPool) LoadBalance(w http.ResponseWriter, r *http.Request) {
	slog.Info("Received request", "method", r.Method, "path", r.URL.Path, "remote", r.RemoteAddr)
	clientID := r.URL.Query().Get("client_id")
	if clientID != "" {
		allowed, err := s.rl.AllowRequest(r.Context(), clientID)
		if err != nil {
			slog.Error("Rate limiting error", "client_id", clientID, "error", err)
			http.Error(w, `{"code": 500, "message": "Internal server error"}`, http.StatusInternalServerError)
			return
		}
		if !allowed {
			slog.Warn("Request rejected due to rate limit", "client_id", clientID)
			http.Error(w, `{"code": 429, "message": "Too many requests"}`, http.StatusTooManyRequests)
			return
		}
	} else {
		slog.Warn("Request without client_id")
	}
	attempts := GetAttemptsFromContext(r)
	if attempts > 3 {
		slog.Error("Max attempts reached, terminating", "remote", r.RemoteAddr, "path", r.URL.Path)
		sendError(w, http.StatusServiceUnavailable, "Service not available")
		return
	}

	backend := s.GetNextBackend()
	if backend == nil {
		slog.Error("No available backends", "remote", r.RemoteAddr, "path", r.URL.Path)
		sendError(w, http.StatusServiceUnavailable, "Service not available")
		return
	}

	backendURL := backend.URL.String()
	slog.Info("Request successfully routed", "backend", backendURL)
	backend.ReverseProxy.ServeHTTP(w, r)
}
