package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/dorik33/cloud/internal/config"
	"github.com/dorik33/cloud/internal/handlers"
	"github.com/dorik33/cloud/internal/loadbalancer"
	"github.com/dorik33/cloud/internal/ratelimit"
	"github.com/dorik33/cloud/internal/store"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level:     slog.LevelDebug,
		AddSource: true,
	}))
	slog.SetDefault(logger)

	cfg := config.LoadConfig("configs/config.yaml")
	slog.Info("", "", cfg.Backends)

	store, err := store.NewConnection(cfg)
	if err != nil {
		slog.Error("Failed to initialize store", "error", err)
		os.Exit(1)
	}
	defer store.Close()

	clientHandler := handlers.NewClientHandler(store.ClientRepository, cfg)
	rateLimiter := ratelimit.NewRateLimiter(store.ClientRepository)
	serverPool := loadbalancer.NewServerPool(rateLimiter)
	for _, backendUrl := range cfg.Backends {
		u, err := url.Parse(backendUrl)
		if err != nil {
			slog.Error("Invalid URL", "url", backendUrl, "error", err)
			os.Exit(1)
		}
		serverPool.AddBackend(u)
	}

	go func() {
		http.ListenAndServe(":8004", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "you redirected to :8004")
		}))
	}()
	go func() {
		http.ListenAndServe(":8001", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "you redirected to :8001")
		}))
	}()

	time.Sleep(1 * time.Second)
	serverPool.HealthCheck()
	serverPool.StartHealthCheck(1 * time.Minute)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	rateLimiter.StartRefillTicker(ctx)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /clients", clientHandler.GetClientsHandler)
	mux.HandleFunc("POST /clients", clientHandler.CreateClientHandler)
	mux.Handle("PUT /clients/{client_id}", http.HandlerFunc(clientHandler.UpdateClientHandler))
	mux.Handle("DELETE /clients/{client_id}", http.HandlerFunc(clientHandler.DeleteClientHandler))
	mux.HandleFunc("/", serverPool.LoadBalance)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%s", cfg.Port),
		Handler: mux,
	}
	slog.Debug("Starting load balancer", "port", cfg.Port)

	if err := server.ListenAndServe(); err != nil {
		slog.Error("Server failed", "error", err)
		os.Exit(1)
	}
}
