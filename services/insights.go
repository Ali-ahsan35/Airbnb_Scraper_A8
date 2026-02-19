package services

import (
	"airbnb-scraper/models"
	"fmt"
	"math"
	"sort"
	"strings"
)

type Report struct {
	TotalListings       int
	AirbnbListings      int
	AveragePrice        float64
	MinPrice            float64
	MaxPrice            float64
	MostExpensive       models.Listing
	TopRated            []models.Listing
	ListingsByLocation  map[string]int
	CleanedListingCount int
}

// GenerateReport cleans the dataset and computes all assignment insights.
func GenerateReport(listings []models.Listing) Report {
	cleaned := cleanListings(listings)

	report := Report{
		TotalListings:       len(cleaned),
		AirbnbListings:      0,
		AveragePrice:        0,
		MinPrice:            0,
		MaxPrice:            0,
		MostExpensive:       models.Listing{},
		TopRated:            nil,
		ListingsByLocation:  make(map[string]int),
		CleanedListingCount: len(cleaned),
	}

	if len(cleaned) == 0 {
		return report
	}

	var (
		priceSum      float64
		priceCount    int
		maxPrice      = -1.0
		minPrice      = math.MaxFloat64
		highestRated  []models.Listing
	)

	for _, l := range cleaned {
		if strings.EqualFold(strings.TrimSpace(l.Platform), "airbnb") {
			report.AirbnbListings++
		}

		location := normalizeLocation(l.Location)
		report.ListingsByLocation[location]++

		if l.Price > 0 {
			priceSum += l.Price
			priceCount++

			if l.Price > maxPrice {
				maxPrice = l.Price
				report.MostExpensive = l
			}
			if l.Price < minPrice {
				minPrice = l.Price
			}
		}

		if l.Rating > 0 {
			highestRated = append(highestRated, l)
		}
	}

	if priceCount > 0 {
		report.AveragePrice = priceSum / float64(priceCount)
		report.MinPrice = minPrice
		report.MaxPrice = maxPrice
	}

	sort.Slice(highestRated, func(i, j int) bool {
		if highestRated[i].Rating == highestRated[j].Rating {
			return highestRated[i].Price > highestRated[j].Price
		}
		return highestRated[i].Rating > highestRated[j].Rating
	})

	if len(highestRated) > 5 {
		highestRated = highestRated[:5]
	}
	report.TopRated = highestRated

	return report
}

func PrintReport(report Report) {
	fmt.Println()
	fmt.Println("Vacation Rental Market Insights")
	fmt.Printf("Total Listings Scraped: %d\n", report.TotalListings)
	fmt.Printf("Airbnb Listings: %d\n", report.AirbnbListings)
	fmt.Printf("Average Price: %.2f\n", report.AveragePrice)
	fmt.Printf("Minimum Price: %.2f\n", report.MinPrice)
	fmt.Printf("Maximum Price: %.2f\n", report.MaxPrice)

	if report.MostExpensive.Title != "" {
		fmt.Println("Most Expensive Property:")
		fmt.Printf("Title: %s\n", report.MostExpensive.Title)
		fmt.Printf("Price: %.2f\n", report.MostExpensive.Price)
		fmt.Printf("Location: %s\n", normalizeLocation(report.MostExpensive.Location))
	}

	fmt.Println("Listings per Location:")
	locations := sortedLocations(report.ListingsByLocation)
	for _, loc := range locations {
		fmt.Printf("%s: %d\n", loc, report.ListingsByLocation[loc])
	}

	fmt.Println("Top 5 Highest Rated Properties:")
	for i, l := range report.TopRated {
		fmt.Printf("%d. %s — %.2f\n", i+1, l.Title, l.Rating)
	}
}

func cleanListings(listings []models.Listing) []models.Listing {
	seen := make(map[string]bool)
	cleaned := make([]models.Listing, 0, len(listings))

	for _, l := range listings {
		l.Title = strings.TrimSpace(l.Title)
		l.URL = strings.TrimSpace(l.URL)
		l.Platform = strings.TrimSpace(strings.ToLower(l.Platform))
		l.Location = strings.TrimSpace(l.Location)

		if l.Title == "" || l.URL == "" {
			continue
		}

		if seen[l.URL] {
			continue
		}

		seen[l.URL] = true
		cleaned = append(cleaned, l)
	}

	return cleaned
}

func normalizeLocation(location string) string {
	location = strings.TrimSpace(location)
	if location == "" {
		return "Unknown"
	}
	return location
}

func sortedLocations(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
