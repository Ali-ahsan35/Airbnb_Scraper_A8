package airbnb

import (
	"airbnb-scraper/config"
	"airbnb-scraper/models"
	"airbnb-scraper/utils"
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/chromedp"
)

type Scraper struct {
	cfg         *config.Config
	allocCtx    context.Context
	allocCancel context.CancelFunc
	seenURLs    map[string]bool
	mu          sync.Mutex
}

func NewScraper(cfg *config.Config) (*Scraper, error) {
	utils.Info("Launching Chrome browser...")
	allocCtx, allocCancel := chromedp.NewExecAllocator(
		context.Background(),
		utils.StealthOpts(cfg.Headless)...,
	)
	utils.Success("Browser ready")
	return &Scraper{
		cfg:         cfg,
		allocCtx:    allocCtx,
		allocCancel: allocCancel,
		seenURLs:    make(map[string]bool),
	}, nil
}

func (s *Scraper) Close() {
	utils.Info("Closing browser...")
	s.allocCancel()
}

func (s *Scraper) isSeen(url string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.seenURLs[url]
}

func (s *Scraper) markSeen(url string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seenURLs[url] = true
}

func (s *Scraper) GetSectionURLs() ([]string, error) {
	utils.Info("Opening homepage to collect section URLs...")

	tabCtx, tabCancel := chromedp.NewContext(s.allocCtx)
	defer tabCancel()

	ctx, cancel := context.WithTimeout(tabCtx, 90*time.Second)
	defer cancel()

	var hrefs []string
	var count int
	var pageTitle string

	err := chromedp.Run(ctx,
		chromedp.Navigate(s.cfg.BaseURL),
		utils.HideWebDriver(),
		
		chromedp.Title(&pageTitle),
		
		chromedp.Sleep(5*time.Second),
		chromedp.Evaluate(`window.scrollTo(0, 800)`, nil),
		chromedp.Sleep(3*time.Second),
		chromedp.Evaluate(`window.scrollTo(0, 1600)`, nil),
		chromedp.Sleep(5*time.Second),
		
		chromedp.Evaluate(`document.querySelectorAll('h2.skp76t2').length`, &count),
		
		chromedp.Evaluate(`
			Array.from(document.querySelectorAll('h2.skp76t2 a'))
				.map(a => {
					let href = a.getAttribute('href');
					return href && href.startsWith('/') ? 'https://www.airbnb.com' + href : (href || '');
				})
				.filter(u => u !== '')
		`, &hrefs),
	)

	if err != nil {
		return nil, fmt.Errorf("homepage error: %w", err)
	}

	utils.Info("Page title: '%s'", pageTitle)
	utils.Info("Found %d h2.skp76t2 elements on page", count)
	utils.Info("Extracted %d hrefs from h2.skp76t2 a", len(hrefs))

	if len(hrefs) == 0 {
		return nil, fmt.Errorf("no section URLs found")
	}

	utils.Success("Found %d section URLs", len(hrefs))
	return hrefs, nil
}

func (s *Scraper) GetPropertyURLsFromSection(sectionURL string) ([]string, error) {
	utils.Info("Getting property URLs from section: %s", sectionURL[:min(60, len(sectionURL))])

	tabCtx, tabCancel := chromedp.NewContext(s.allocCtx)
	defer tabCancel()

	ctx, cancel := context.WithTimeout(tabCtx, s.cfg.RequestTimeout)
	defer cancel()

	var urls []string

	err := chromedp.Run(ctx,
		chromedp.Navigate(sectionURL),
		utils.HideWebDriver(),
		chromedp.WaitVisible(`[data-testid="listing-card-title"]`, chromedp.ByQuery),
		chromedp.Sleep(3*time.Second),

		chromedp.Evaluate(`
			Array.from(document.querySelectorAll('[data-testid="listing-card-title"]'))
				.slice(0, 3)
				.map(titleEl => {
					let card = titleEl.parentElement.parentElement.parentElement;
					let linkEl = card.querySelector('a.l1ovpqvx');
					return linkEl ? linkEl.href : '';
				})
				.filter(u => u !== '')
		`, &urls),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get property URLs: %w", err)
	}

	utils.Success("Got %d property URLs from section", len(urls))
	return urls, nil
}

func (s *Scraper) ScrapePropertyPage(propertyURL string) (models.Listing, error) {
	if s.isSeen(propertyURL) {
		return models.Listing{}, nil
	}

	utils.Info("Visiting property: %s", propertyURL[:min(70, len(propertyURL))])
	utils.RandomDelay(s.cfg.MinDelay, s.cfg.MaxDelay)

	var listing models.Listing
	var err error

	err = utils.Retry(s.cfg.MaxRetries, func() error {
		listing, err = s.extractFromPropertyPage(propertyURL)
		return err
	})

	if err != nil {
		return models.Listing{}, err
	}

	s.markSeen(propertyURL)
	utils.Success("✓ %s | $%.0f | %.2f★", truncate(listing.Title, 30), listing.Price, listing.Rating)
	return listing, nil
}

func (s *Scraper) extractFromPropertyPage(propertyURL string) (models.Listing, error) {
	tabCtx, tabCancel := chromedp.NewContext(s.allocCtx)
	defer tabCancel()

	ctx, cancel := context.WithTimeout(tabCtx, s.cfg.RequestTimeout)
	defer cancel()

	var title, price, location, rating, description string

	err := chromedp.Run(ctx,
		chromedp.Navigate(propertyURL),
		utils.HideWebDriver(),
		
		chromedp.WaitVisible(`h1`, chromedp.ByQuery),
		chromedp.Sleep(4*time.Second),

		chromedp.Evaluate(`
			document.querySelector('h1')?.textContent.trim() || ''
		`, &title),

		chromedp.Evaluate(`
			document.querySelector('.u174bpcy')?.textContent.trim() || ''
		`, &price),

		chromedp.Evaluate(`
			(() => {
				let h2 = document.querySelector('h2.hpipapi');
				if (!h2) return '';
				let text = h2.textContent.trim();
				let match = text.match(/in (.+?)$/);
				return match ? match[1] : text;
			})()
		`, &location),

		chromedp.Evaluate(`
			(() => {
				const normalize = (txt) => {
					if (!txt) return '';
					const m = txt.match(/([0-9]+(?:\.[0-9]+)?)/);
					return m ? m[1] : '';
				};

				// Primary location used by many Airbnb PDP layouts
				const banner = document.querySelector('[data-testid="pdp-reviews-highlight-banner-host-rating"]');
				if (banner) {
					const ratingDiv = banner.querySelector('div[aria-hidden="true"]');
					const fromBanner = normalize(ratingDiv ? ratingDiv.textContent.trim() : '');
					if (fromBanner) return fromBanner;
				}

				// Alternative location: screen-reader text like "Rated 4.81 out of 5 stars."
				const ratedTextEl = Array.from(document.querySelectorAll('span')).find(el =>
					/Rated\s+[0-9]+(?:\.[0-9]+)?\s+out of 5 stars\./i.test(el.textContent || '')
				);
				const fromRatedText = normalize(ratedTextEl ? ratedTextEl.textContent.trim() : '');
				if (fromRatedText) return fromRatedText;

				// Alternative visual element that often holds the numeric rating
				const altDiv = document.querySelector('div.rmtgcc3[aria-hidden="true"]');
				const fromAltDiv = normalize(altDiv ? altDiv.textContent.trim() : '');
				if (fromAltDiv) return fromAltDiv;

				return '';
			})()
		`, &rating),

		chromedp.Evaluate(`
			(() => {
				const clean = (txt) => (txt || '').replace(/\s+/g, ' ').trim();

				// Preferred source from DESCRIPTION_DEFAULT block
				const preferred = document.querySelector(
					'[data-section-id="DESCRIPTION_DEFAULT"] span.l1h825yc, [data-plugin-in-point-id="DESCRIPTION_DEFAULT"] span.l1h825yc'
				);
				const fromPreferred = clean(preferred ? (preferred.innerText || preferred.textContent) : '');
				if (fromPreferred) return fromPreferred;

				// Fallback to section container text
				const section = document.querySelector(
					'[data-section-id="DESCRIPTION_DEFAULT"], [data-plugin-in-point-id="DESCRIPTION_DEFAULT"]'
				);
				const fromSection = clean(section ? (section.innerText || section.textContent) : '');
				if (fromSection) return fromSection;

				// Legacy fallback selector
				return clean(document.querySelector('[data-testid="listing-page-summary"]')?.textContent || '');
			})()
		`, &description),
	)

	if err != nil {
		return models.Listing{}, fmt.Errorf("chromedp failed: %w", err)
	}

	return models.Listing{
		Platform:    "airbnb",
		Title:       strings.TrimSpace(title),
		RawPrice:    price,
		Price:       parsePrice(price),
		Location:    strings.TrimSpace(location),
		Rating:      parseRating(rating),
		URL:         propertyURL,
		Description: truncate(strings.TrimSpace(description), 200),
	}, nil
}

func parsePrice(raw string) float64 {
	cleaned := strings.NewReplacer("$", "", ",", "", "€", "", "£", "", "RM", "").Replace(raw)
	parts := strings.Fields(cleaned)
	if len(parts) == 0 {
		return 0
	}
	var v float64
	fmt.Sscanf(parts[0], "%f", &v)
	return v
}

func parseRating(raw string) float64 {
	if raw == "" {
		return 0
	}
	var v float64
	fmt.Sscanf(raw, "%f", &v)
	if v < 0 || v > 5 {
		return 0
	}
	return v
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
