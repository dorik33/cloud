package models

import "time"

type Client struct {
	ClientID   string    `json:"client_id"`
	Capacity   int       `json:"capacity"`
	RatePerSec int       `json:"rate_per_sec"`
	Tokens     int       `json:"tokens"`
	LastRefill time.Time `json:"last_refill"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type CreateClient struct {
	ClientID   string `json:"client_id"`
	Capacity   int    `json:"capacity"`
	RatePerSec int    `json:"rate_per_sec"`
}

type UpdateClient struct {
	Capacity   int `json:"capacity"`
	RatePerSec int `json:"rate_per_sec"`
}
