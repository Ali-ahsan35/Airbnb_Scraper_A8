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
	DBHost         string
	DBPort         int
	DBUser         string
	DBPassword     string
	DBName         string
	DBSSLMode      string
}

func DefaultConfig() *Config {
	return &Config{
		BaseURL:        "https://www.airbnb.com/",
		MaxPages:       5,
		MaxWorkers:     3,
		RequestTimeout: 60 * time.Second,
		MinDelay:       3 * time.Second,
		MaxDelay:       7 * time.Second,
		MaxRetries:     3,
		Headless:       true,
		CSVPath:        "output/listings.csv",
		DBHost:         "localhost",
		DBPort:         5433,
		DBUser:         "postgres",
		DBPassword:     "postgres",
		DBName:         "airbnb_scraper",
		DBSSLMode:      "disable",
	}
}
