package store

import (
	"context"
	"fmt"
	"time"

	"github.com/dorik33/cloud/internal/models"
	"github.com/jackc/pgx/v5"
)

type ClientRepository interface {
	Create(ctx context.Context, client *models.Client) error
	GetByID(ctx context.Context, clientID string) (*models.Client, error)
	GetByIDForUpdate(ctx context.Context, clientID string) (*models.Client, error)
	GetAllClients(ctx context.Context) ([]*models.Client, error)
	Update(ctx context.Context, client *models.Client) error
	Delete(ctx context.Context, clientID string) error
}

type clientRepository struct {
	store *Store
}

func (r *clientRepository) Create(ctx context.Context, client *models.Client) error {
	query := `
		INSERT INTO clients (client_id, capacity, rate_per_sec, tokens, last_refill, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.store.pool.Exec(ctx, query,
		client.ClientID, client.Capacity, client.RatePerSec, client.Tokens,
		client.LastRefill, client.CreatedAt, client.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	return nil
}

func (r *clientRepository) GetByID(ctx context.Context, clientID string) (*models.Client, error) {
	query := `
		SELECT client_id, capacity, rate_per_sec, tokens, last_refill, created_at, updated_at
		FROM clients WHERE client_id = $1
	`
	client := &models.Client{}
	err := r.store.pool.QueryRow(ctx, query, clientID).Scan(
		&client.ClientID, &client.Capacity, &client.RatePerSec, &client.Tokens,
		&client.LastRefill, &client.CreatedAt, &client.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get client %s: %w", clientID, err)
	}
	return client, nil
}

func (r *clientRepository) GetByIDForUpdate(ctx context.Context, clientID string) (*models.Client, error) {
	client := &models.Client{}
	err := r.store.pool.QueryRow(ctx,
		`SELECT client_id, capacity, rate_per_sec, tokens, last_refill, created_at, updated_at
		FROM clients WHERE client_id = $1 FOR UPDATE`,
		clientID).Scan(
		&client.ClientID, &client.Capacity, &client.RatePerSec, &client.Tokens,
		&client.LastRefill, &client.CreatedAt, &client.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return client, nil
}

func (r *clientRepository) GetAllClients(ctx context.Context) ([]*models.Client, error) {
	query := `
		SELECT client_id, capacity, rate_per_sec, tokens, last_refill, created_at, updated_at
		FROM clients
	`
	rows, err := r.store.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get all clients: %w", err)
	}
	defer rows.Close()

	var clients []*models.Client
	for rows.Next() {
		client := &models.Client{}
		err := rows.Scan(
			&client.ClientID, &client.Capacity, &client.RatePerSec, &client.Tokens,
			&client.LastRefill, &client.CreatedAt, &client.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan client: %w", err)
		}
		clients = append(clients, client)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating clients: %w", err)
	}

	return clients, nil
}

func (r *clientRepository) Update(ctx context.Context, client *models.Client) error {
	query := `
		UPDATE clients
		SET capacity = $2, rate_per_sec = $3, tokens = $4, last_refill = $5, updated_at = CURRENT_TIMESTAMP
		WHERE client_id = $1
		RETURNING updated_at
	`
	var updatedAt time.Time
	err := r.store.pool.QueryRow(ctx, query,
		client.ClientID, client.Capacity, client.RatePerSec, client.Tokens,
		client.LastRefill).Scan(&updatedAt)
	if err == pgx.ErrNoRows {
		return fmt.Errorf("client with id %s not found", client.ClientID)
	}
	if err != nil {
		return fmt.Errorf("failed to update client %s: %w", client.ClientID, err)
	}
	client.UpdatedAt = updatedAt
	return nil
}

func (r *clientRepository) Delete(ctx context.Context, clientID string) error {
	query := `DELETE FROM clients WHERE client_id = $1`
	_, err := r.store.pool.Exec(ctx, query, clientID)
	if err != nil {
		return fmt.Errorf("failed to delete client %s: %w", clientID, err)
	}
	return nil
}
