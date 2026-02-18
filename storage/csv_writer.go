package storage

import (
	"airbnb-scraper/models"
	"airbnb-scraper/utils"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

// CSVWriter saves listings to a CSV file.
// Used on Day 1 as temporary storage before PostgreSQL is added on Day 2.
type CSVWriter struct {
	path string
}

func NewCSVWriter(path string) *CSVWriter {
	return &CSVWriter{path: path}
}

// Write saves all listings to the CSV file.
// Creates the output directory if it does not exist.
//
// CSV columns: platform, title, price, raw_price, location, rating, url
func (w *CSVWriter) Write(listings []models.Listing) error {
	if len(listings) == 0 {
		utils.Warn("No listings to write")
		return nil
	}

	// Create output directory if needed (e.g. "output/" folder)
	if err := os.MkdirAll(filepath.Dir(w.path), 0755); err != nil {
		return fmt.Errorf("could not create output dir: %w", err)
	}

	file, err := os.Create(w.path)
	if err != nil {
		return fmt.Errorf("could not create file: %w", err)
	}
	defer file.Close()

	// csv.NewWriter handles quoting, commas inside fields, line endings
	writer := csv.NewWriter(file)
	defer writer.Flush() // IMPORTANT — must flush or data stays in buffer

	// Header row
	writer.Write([]string{"platform", "title", "price", "raw_price", "location", "rating", "url", "description"})

	// One row per listing
	for _, l := range listings {
		writer.Write([]string{
			l.Platform,
			l.Title,
			strconv.FormatFloat(l.Price, 'f', 2, 64),
			l.RawPrice,
			l.Location,
			strconv.FormatFloat(l.Rating, 'f', 2, 64),
			l.URL,
			l.Description,
		})
	}

	// Check if any writes failed
	if err := writer.Error(); err != nil {
		return fmt.Errorf("csv write error: %w", err)
	}

	utils.Success("Saved %d listings → %s", len(listings), w.path)
	return nil
}