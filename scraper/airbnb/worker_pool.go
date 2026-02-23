package airbnb

import (
	"airbnb-scraper/config"
	"airbnb-scraper/models"
	"airbnb-scraper/utils"
	"sync"
)

type WorkerPool struct {
	scraper *Scraper
	cfg     *config.Config
	jobs    chan string
	results chan models.ScrapeResult
	wg      sync.WaitGroup
}

func NewWorkerPool(scraper *Scraper, cfg *config.Config) *WorkerPool {
	return &WorkerPool{
		scraper: scraper,
		cfg:     cfg,
	}
}

func (p *WorkerPool) Run() []models.Listing {
	utils.Info("Collecting section URLs")

	sectionURLs, err := p.scraper.GetSectionURLs()
	if err != nil {
		utils.Error("Failed to get section URLs: %v", err)
		return nil
	}

	if len(sectionURLs) < p.cfg.MaxPages {
		utils.Warn("Only %d sections found, need %d", len(sectionURLs), p.cfg.MaxPages)
		p.cfg.MaxPages = len(sectionURLs)
	}

	utils.Info("Processing up to %d sections", p.cfg.MaxPages)

	var allListings []models.Listing

	for pageNum := 1; pageNum <= p.cfg.MaxPages; pageNum++ {
		sectionURL := sectionURLs[pageNum-1]

		propertyURLs, err := p.scraper.GetPropertyURLsFromSection(sectionURL)
		if err != nil {
			utils.Error("Page %d failed: %v", pageNum, err)
			continue
		}

		if len(propertyURLs) == 0 {
			continue
		}

		utils.Info("Scraping property details for section %d", pageNum)
		sectionListings := p.scrapeProperties(propertyURLs)
		allListings = append(allListings, sectionListings...)
	}

	if len(allListings) == 0 {
		utils.Error("No listings scraped from any section")
		return nil
	}

	utils.Success("Total listings scraped from all sections: %d", len(allListings))
	return allListings
}

func (p *WorkerPool) scrapeProperties(propertyURLs []string) []models.Listing {
	p.jobs = make(chan string, len(propertyURLs))
	p.results = make(chan models.ScrapeResult, len(propertyURLs))

	workerCount := p.cfg.MaxWorkers
	if len(propertyURLs) < workerCount {
		workerCount = len(propertyURLs)
	}

	p.wg.Add(workerCount)
	for i := 1; i <= workerCount; i++ {
		go p.worker(i)
	}

	for _, url := range propertyURLs {
		p.jobs <- url
	}
	close(p.jobs)

	go func() {
		p.wg.Wait()
		close(p.results)
	}()

	return p.collect()
}

func (p *WorkerPool) worker(id int) {
	defer p.wg.Done()

	for propertyURL := range p.jobs {
		listing, err := p.scraper.ScrapePropertyPage(propertyURL)

		p.results <- models.ScrapeResult{
			Listings: []models.Listing{listing},
			Error:    err,
		}
	}
}

func (p *WorkerPool) collect() []models.Listing {
	var all []models.Listing
	failed := 0

	for result := range p.results {
		if result.Error != nil {
			utils.Error("Property failed: %v", result.Error)
			failed++
			continue
		}
		if len(result.Listings) > 0 && result.Listings[0].Title != "" {
			all = append(all, result.Listings...)
		}
	}

	utils.Success("Properties scraped: %d | Failed: %d", len(all), failed)
	return all
}
