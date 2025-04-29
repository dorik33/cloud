package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/dorik33/cloud/internal/config"
	"github.com/dorik33/cloud/internal/models"
	"github.com/dorik33/cloud/internal/store"
)

type ClientHandler struct {
	repo store.ClientRepository
	cfg  *config.Config
}

func NewClientHandler(repo store.ClientRepository, cfg *config.Config) *ClientHandler {
	return &ClientHandler{repo: repo, cfg: cfg}
}

func (h *ClientHandler) GetClientsHandler(w http.ResponseWriter, r *http.Request) {
	slog.Debug("Handling get clients request", "method", r.Method, "path", r.URL.Path)

	clients, err := h.repo.GetAllClients(r.Context())
	if err != nil {
		slog.Error("Failed to get clients", "error", err)
		sendError(w, http.StatusInternalServerError, "Error with server")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(clients)
}

func (h *ClientHandler) CreateClientHandler(w http.ResponseWriter, r *http.Request) {
	slog.Debug("Handling create client request", "method", r.Method, "path", r.URL.Path)

	req := models.CreateClient{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Error("Failed to decode request body", "error", err)
		sendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.ClientID == "" {
		slog.Error("Client ID is required")
		sendError(w, http.StatusBadRequest, "Client ID is required")
		return
	}
	if req.Capacity <= 0 {
		req.Capacity = h.cfg.RateLimit.Capacity
		slog.Debug("Using default capacity", "capacity", req.Capacity)
	}
	if req.RatePerSec <= 0 {
		req.RatePerSec = h.cfg.RateLimit.Rate
		slog.Debug("Using default rate per second", "rate_per_sec", req.RatePerSec)
	}

	client := &models.Client{
		ClientID:   req.ClientID,
		Capacity:   req.Capacity,
		RatePerSec: req.RatePerSec,
		Tokens:     req.Capacity,
		LastRefill: time.Now(),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if err := h.repo.Create(r.Context(), client); err != nil {
		slog.Error("Failed to create client", "client_id", req.ClientID, "error", err)
		sendError(w, http.StatusInternalServerError, "Failed to create client")
		return
	}

	slog.Info("Client created", "client_id", client.ClientID, "capacity", client.Capacity, "rate_per_sec", client.RatePerSec)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(client)
}

func (h *ClientHandler) UpdateClientHandler(w http.ResponseWriter, r *http.Request) {
	clientID := r.PathValue("client_id")
	slog.Debug("Updating client", "client_id", clientID)

	var req struct {
		Capacity   int `json:"capacity"`
		RatePerSec int `json:"rate_per_sec"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Error("Failed to decode request body", "client_id", clientID, "error", err)
		sendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Capacity <= 0 {
		slog.Error("Capacity must be greater than 0", "client_id", clientID, "capacity", req.Capacity)
		sendError(w, http.StatusBadRequest, "Capacity must be greater than 0")
		return
	}
	if req.RatePerSec <= 0 {
		slog.Error("Rate per second must be greater than 0", "client_id", clientID, "rate_per_sec", req.RatePerSec)
		sendError(w, http.StatusBadRequest, "Rate per second must be greater than 0")
		return
	}

	client, err := h.repo.GetByID(r.Context(), clientID)
	if err != nil {
		slog.Error("Failed to get client", "client_id", clientID, "error", err)
		sendError(w, http.StatusInternalServerError, "Failed to get client")
		return
	}
	if client == nil {
		slog.Error("Client not found", "client_id", clientID)
		sendError(w, http.StatusNotFound, fmt.Sprintf("Client with id %s not found", clientID))
		return
	}

	client.Capacity = req.Capacity
	client.RatePerSec = req.RatePerSec
	if client.Tokens > req.Capacity {
		client.Tokens = req.Capacity
	}
	client.LastRefill = time.Now()

	if err := h.repo.Update(r.Context(), client); err != nil {
		slog.Error("Failed to update client", "client_id", clientID, "error", err)
		sendError(w, http.StatusInternalServerError, "Failed to update client")
		return
	}

	slog.Info("Client updated", "client_id", client.ClientID, "capacity", client.Capacity, "rate_per_sec", client.RatePerSec)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(client)
}

func (h *ClientHandler) DeleteClientHandler(w http.ResponseWriter, r *http.Request) {
	clientID := r.PathValue("client_id")
	slog.Debug("Deleting client", "client_id", clientID)

	if err := h.repo.Delete(r.Context(), clientID); err != nil {
		slog.Error("Failed to delete client", "client_id", clientID, "error", err)
		sendError(w, http.StatusInternalServerError, "Failed to delete client")
		return
	}

	slog.Info("Client deleted", "client_id", clientID)
	w.WriteHeader(http.StatusNoContent)
}
