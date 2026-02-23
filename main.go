package main

import (
	"airbnb-scraper/config"
	"airbnb-scraper/models"
	"airbnb-scraper/scraper/airbnb"
	"airbnb-scraper/services"
	"airbnb-scraper/storage"
	"airbnb-scraper/utils"
	"fmt"
	"os"
)

func main() {
	cfg := config.DefaultConfig()
	utils.Info("Scraper starting | pages=%d workers=%d delay=%v-%v",
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

	cleanedListings := services.CleanListings(listings)
	if len(cleanedListings) == 0 {
		utils.Warn("No valid listings after cleaning.")
		os.Exit(0)
	}

	writer := storage.NewCSVWriter(cfg.CSVPath)
	if err := writer.Write(cleanedListings); err != nil {
		utils.Error("Failed to save CSV: %v", err)
		os.Exit(1)
	}

	pgWriter, err := storage.NewPostgresWriter(cfg)
	if err != nil {
		utils.Error("Failed to connect PostgreSQL: %v", err)
		os.Exit(1)
	}
	defer pgWriter.Close()

	if err := pgWriter.EnsureSchema(); err != nil {
		utils.Error("Failed to ensure PostgreSQL schema: %v", err)
		os.Exit(1)
	}

	if err := pgWriter.WriteBatch(cleanedListings); err != nil {
		utils.Error("Failed to save listings to PostgreSQL: %v", err)
		os.Exit(1)
	}
	utils.Success("Saved %d cleaned listings to PostgreSQL", len(cleanedListings))

	printSummary(cleanedListings)
	report := services.GenerateReport(cleanedListings)
	services.PrintReport(report)
}

func printSummary(listings []models.Listing) {
	fmt.Println()
	fmt.Println("╔══════════════════════════════════════════════╗")
	fmt.Println("║                SCRAPE COMPLETE               ║")
	fmt.Println("╠══════════════════════════════════════════════╣")
	fmt.Printf( "║  Total listings : %-26d║\n", len(listings))
	fmt.Println("╚══════════════════════════════════════════════╝")
	fmt.Println()
}
