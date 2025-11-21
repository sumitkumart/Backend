package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config captures application level settings loaded from the environment.
type Config struct {
	HTTPPort    string
	DatabaseURL string
	Fees        FeeConfig
	Price       PriceConfig
}

type FeeConfig struct {
	BrokerageBps int
	TaxBps       int
}

type PriceConfig struct {
	JobInterval      time.Duration
	RandomFloorPrice float64
	RandomCeilPrice  float64
}

// Load parses environment variables into Config and falls back to sensible defaults
// so the server can boot without additional flags.
func Load() (*Config, error) {
	_ = godotenv.Load(".env")

	cfg := &Config{
		HTTPPort:    getEnv("PORT", "8080"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
		Fees: FeeConfig{
			BrokerageBps: getInt("BROKERAGE_BPS", 40), // 0.40%
			TaxBps:       getInt("TAX_BPS", 35),       // 0.35%
		},
		Price: PriceConfig{
			JobInterval:      getDuration("PRICE_JOB_INTERVAL", time.Hour),
			RandomFloorPrice: getFloat("PRICE_RANDOM_FLOOR", 1200.0),
			RandomCeilPrice:  getFloat("PRICE_RANDOM_CEIL", 3200.0),
		},
	}

	if cfg.DatabaseURL == "" {
		return nil, errors.New("DATABASE_URL is required")
	}

	if cfg.Price.RandomFloorPrice <= 0 || cfg.Price.RandomCeilPrice <= 0 {
		return nil, errors.New("invalid random price bounds configured")
	}

	if cfg.Price.RandomCeilPrice <= cfg.Price.RandomFloorPrice {
		return nil, errors.New("PRICE_RANDOM_CEIL must be greater than PRICE_RANDOM_FLOOR")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getInt(key string, fallback int) int {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(val)
	if err != nil {
		return fallback
	}
	return parsed
}

func getFloat(key string, fallback float64) float64 {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	parsed, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return fallback
	}
	return parsed
}

func getDuration(key string, fallback time.Duration) time.Duration {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(val)
	if err != nil {
		return fallback
	}
	return parsed
}

// FeesAsPercent returns brokerage and tax percentages as decimals for convenience.
func (cfg *FeeConfig) FeesAsPercent() (brokerage float64, tax float64) {
	return bpsToPercent(cfg.BrokerageBps), bpsToPercent(cfg.TaxBps)
}

func bpsToPercent(bps int) float64 {
	return float64(bps) / 10000
}

func (cfg FeeConfig) String() string {
	return fmt.Sprintf("brokerage=%dbps tax=%dbps", cfg.BrokerageBps, cfg.TaxBps)
}
