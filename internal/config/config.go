package config

import (
	"log/slog"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Port      string          `yaml:"port"`
	Backends  []string        `yaml:"backends"`
	RateLimit RateLimitConfig `yaml:"rate_limit"`
	DBConnStr string          `yaml:"db_conn_str"`
}

type RateLimitConfig struct {
	Capacity int `yaml:"default_capacity"`
	Rate     int `yaml:"default_rate"`
}

func LoadConfig(path string) *Config {
	var cfg Config
	err := cleanenv.ReadConfig(path, &cfg)
	if err != nil {
		slog.Error("Cannot read config", "error", err)
		os.Exit(1)
	}
	return &cfg
}
