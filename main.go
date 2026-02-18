package main

import (
	"airbnb-scraper/config"
	"airbnb-scraper/models"
	"airbnb-scraper/scraper/airbnb"
	"airbnb-scraper/storage"
	"airbnb-scraper/utils"
	"fmt"
	"os"
)

func main() {
	utils.Section("Airbnb Scraper — Day 1")

	cfg := config.DefaultConfig()
	utils.Info("Pages: %d  |  Workers: %d  |  Delay: %v-%v",
		cfg.MaxPages, cfg.MaxWorkers, cfg.MinDelay, cfg.MaxDelay)

	scraper, err := airbnb.NewScraper(cfg)
	if err != nil {
		utils.Error("Could not start scraper: %v", err)
		os.Exit(1)
	}
	defer scraper.Close()

	pool := airbnb.NewWorkerPool(scraper, cfg)
	listings := pool.Run()

	if len(listings) == 0 {
		utils.Warn("No listings scraped.")
		os.Exit(0)
	}

	writer := storage.NewCSVWriter(cfg.CSVPath)
	if err := writer.Write(listings); err != nil {
		utils.Error("Failed to save CSV: %v", err)
		os.Exit(1)
	}

	printSummary(listings)
}

func printSummary(listings []models.Listing) {
	fmt.Println()
	fmt.Println("╔══════════════════════════════════════════════╗")
	fmt.Println("║                  SCRAPE COMPLETE             ║")
	fmt.Println("╠══════════════════════════════════════════════╣")
	fmt.Printf( "║  Total listings : %-26d║\n", len(listings))
	fmt.Println("╚══════════════════════════════════════════════╝")
	fmt.Println()
}