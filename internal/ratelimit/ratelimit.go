package ratelimit

import (
	"context"
	"log/slog"
	"time"

	"github.com/dorik33/cloud/internal/store"
)

type RateLimiter struct {
	repo store.ClientRepository
}

func NewRateLimiter(repo store.ClientRepository) *RateLimiter {
	return &RateLimiter{repo: repo}
}

func (rl *RateLimiter) AllowRequest(ctx context.Context, clientID string) (bool, error) {
	client, err := rl.repo.GetByIDForUpdate(ctx, clientID)
	if err != nil {
		slog.Error("Failed to get client for rate limiting", "client_id", clientID, "error", err)
		return false, err
	}
	if client == nil {
		slog.Error("Client not found for rate limiting", "client_id", clientID)
		return false, nil
	}

	if client.Tokens < 10 {
		slog.Debug("Rate limit exceeded", "client_id", clientID, "tokens", client.Tokens)
		return false, nil
	}

	client.Tokens -= 10
	client.LastRefill = time.Now()

	if err := rl.repo.Update(ctx, client); err != nil {
		slog.Error("Failed to update client tokens", "client_id", clientID, "error", err)
		return false, err
	}

	slog.Debug("Request allowed", "client_id", clientID, "tokens", client.Tokens)
	return true, nil
}

func (rl *RateLimiter) RefillAllTokens(ctx context.Context) error {
	clients, err := rl.repo.GetAllClients(ctx)
	if err != nil {
		slog.Error("Failed to get all clients for token refill", "error", err)
		return err
	}

	for _, client := range clients {
		client, err := rl.repo.GetByIDForUpdate(ctx, client.ClientID)
		if err != nil {
			slog.Error("Failed to get client for token refill", "client_id", client.ClientID, "error", err)
			continue
		}
		if client == nil {
			slog.Warn("Client not found during token refill", "client_id", client.ClientID)
			continue
		}
		if client.Tokens < client.Capacity {
			client.Tokens += client.RatePerSec
			rl.repo.Update(ctx, client)
		}
	}
	return nil
}

func (rl *RateLimiter) StartRefillTicker(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	go func() {
		for {
			select {
			case <-ticker.C:
				rl.RefillAllTokens(ctx)
			case <-ctx.Done():
				ticker.Stop()

				return
			}
		}
	}()
	slog.Info("Token refill ticker started")
}
