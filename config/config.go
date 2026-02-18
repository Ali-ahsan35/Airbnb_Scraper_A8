package config

import "time"

type Config struct {
	BaseURL        string
	MaxPages       int
	MaxWorkers     int
	RequestTimeout time.Duration
	MinDelay       time.Duration
	MaxDelay       time.Duration
	MaxRetries     int
	Headless       bool
	CSVPath        string
}

func DefaultConfig() *Config {
	return &Config{
		BaseURL:        "https://www.airbnb.com/",
		MaxPages:       9,
		MaxWorkers:     1,
		RequestTimeout: 60 * time.Second,
		MinDelay:       3 * time.Second,
		MaxDelay:       7 * time.Second,
		MaxRetries:     3,
		Headless:       false,
		CSVPath:        "output/listings.csv",
	}
}
