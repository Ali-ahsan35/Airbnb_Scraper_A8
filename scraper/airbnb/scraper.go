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

func (s *Scraper) markSeenIfNew(url string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.seenURLs[url] {
		return false
	}
	s.seenURLs[url] = true
	return true
}

func (s *Scraper) GetSectionURLs() ([]string, error) {
	utils.Info("Opening homepage to collect section URLs...")

	tabCtx, tabCancel := chromedp.NewContext(s.allocCtx)
	defer tabCancel()

	ctx, cancel := context.WithTimeout(tabCtx, 90*time.Second)
	defer cancel()

	var hrefs []string

	err := chromedp.Run(ctx,
		chromedp.Navigate(s.cfg.BaseURL),
		utils.HideWebDriver(),
		chromedp.Sleep(5*time.Second),
		chromedp.Evaluate(`window.scrollTo(0, document.body.scrollHeight * 0.35)`, nil),
		chromedp.Sleep(3*time.Second),
		chromedp.Evaluate(`window.scrollTo(0, document.body.scrollHeight * 0.7)`, nil),
		chromedp.Sleep(4*time.Second),
		chromedp.Evaluate(`(() => {
			const toAbs = (href) => {
				if (!href) return '';
				if (href.startsWith('http')) return href;
				if (href.startsWith('/')) return 'https://www.airbnb.com' + href;
				return '';
			};

			// Prefer destination/search section links (e.g. /s/City/homes?...).
			const candidates = Array.from(document.querySelectorAll('a[href*="/s/"]'))
				.map(a => toAbs(a.getAttribute('href')))
				.filter(u =>
					u.includes('/s/') &&
					(u.includes('/homes') || u.includes('search_mode=')) &&
					!u.includes('/rooms/')
				);

			// Stable unique list while preserving order.
			const seen = new Set();
			const unique = [];
			for (const u of candidates) {
				if (!seen.has(u)) {
					seen.add(u);
					unique.push(u);
				}
			}
			return unique;
		})()`, &hrefs),
	)

	if err != nil {
		return nil, fmt.Errorf("homepage error: %w", err)
	}

	if len(hrefs) == 0 {
		return nil, fmt.Errorf("no section URLs found")
	}

	utils.Success("Found %d section URLs", len(hrefs))
	return hrefs, nil
}

func (s *Scraper) GetPropertyURLsFromSection(sectionURL string) ([]string, error) {
	tabCtx, tabCancel := chromedp.NewContext(s.allocCtx)
	defer tabCancel()

	ctx, cancel := context.WithTimeout(tabCtx, s.cfg.RequestTimeout)
	defer cancel()

	var urls []string
	seen := make(map[string]bool)

	err := chromedp.Run(ctx,
		chromedp.Navigate(sectionURL),
		utils.HideWebDriver(),
		chromedp.WaitVisible(`[data-testid="listing-card-title"]`, chromedp.ByQuery),
		chromedp.Sleep(3*time.Second),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get property URLs: %w", err)
	}

	extractTopTwo := func() ([]string, error) {
		var pageURLs []string
		err := chromedp.Run(ctx, chromedp.Evaluate(`
			Array.from(document.querySelectorAll('[data-testid="listing-card-title"]'))
				.slice(0, 5)
				.map(titleEl => {
					const card = titleEl.closest('div[itemprop="itemListElement"]') || titleEl.closest('div');
					if (!card) return '';
					const linkEl = card.querySelector('a[href*="/rooms/"]');
					if (!linkEl) return '';
					const href = linkEl.getAttribute('href') || '';
					return href.startsWith('/') ? 'https://www.airbnb.com' + href : href;
				})
				.filter(u => u !== '')
		`, &pageURLs))
		return pageURLs, err
	}

	addUnique := func(candidates []string) {
		for _, u := range candidates {
			if !seen[u] {
				seen[u] = true
				urls = append(urls, u)
			}
		}
	}

	page1URLs, err := extractTopTwo()
	if err != nil {
		return nil, fmt.Errorf("failed to parse page 1 property URLs: %w", err)
	}
	addUnique(page1URLs)

	var movedToPage2 bool
	err = chromedp.Run(ctx,
		chromedp.Evaluate(`(() => {
			const selectors = [
				'a[aria-label*="Next"]',
				'button[aria-label*="Next"]',
				'a[aria-label*="next"]',
				'button[aria-label*="next"]'
			];
			for (const sel of selectors) {
				const el = document.querySelector(sel);
				if (el) {
					el.click();
					return true;
				}
			}
			return false;
		})()`, &movedToPage2),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to move to page 2: %w", err)
	}

	if movedToPage2 {
		err = chromedp.Run(ctx,
			chromedp.Sleep(4*time.Second),
			chromedp.WaitVisible(`[data-testid="listing-card-title"]`, chromedp.ByQuery),
		)
		if err != nil {
			return nil, fmt.Errorf("page 2 did not load: %w", err)
		}

		page2URLs, err := extractTopTwo()
		if err != nil {
			return nil, fmt.Errorf("failed to parse page 2 property URLs: %w", err)
		}
		addUnique(page2URLs)
	} else {
		utils.Warn("Could not find Next button for section pagination; using only page 1")
	}

	utils.Success("Got %d property URLs from section", len(urls))
	return urls, nil
}

func (s *Scraper) ScrapePropertyPage(propertyURL string) (models.Listing, error) {
	if !s.markSeenIfNew(propertyURL) {
		return models.Listing{}, nil
	}

	utils.RandomDelay(s.cfg.MinDelay, s.cfg.MaxDelay)

	var listing models.Listing
	var err error

	err = utils.Retry(s.cfg.MaxRetries, func() error {
		listing, err = s.extractFromPropertyPage(propertyURL)
		return err
	})

	if err != nil {
		s.mu.Lock()
		delete(s.seenURLs, propertyURL)
		s.mu.Unlock()
		return models.Listing{}, err
	}

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
			(() => {
				const firstMoney = (txt) => {
					if (!txt) return '';
					const m = txt.match(/\$?\s*([0-9][0-9,]*(?:\.[0-9]+)?)/);
					return m ? ('$' + m[1]) : '';
				};

				const ariaPrice = Array.from(document.querySelectorAll('[aria-label]')).find(el =>
					/\$\s*[0-9]/.test(el.getAttribute('aria-label') || '') &&
					/for\s+[0-9]+\s+nights?/i.test(el.getAttribute('aria-label') || '')
				);
				const fromAria = firstMoney(ariaPrice ? ariaPrice.getAttribute('aria-label') : '');
				if (fromAria) return fromAria;

				const visibleTotal = document.querySelector('span.u1opajno');
				const fromVisibleTotal = firstMoney(visibleTotal ? visibleTotal.textContent : '');
				if (fromVisibleTotal) return fromVisibleTotal;

				const fromOld = firstMoney(document.querySelector('.u174bpcy')?.textContent || '');
				if (fromOld) return fromOld;

				return '';
			})()
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

				const banner = document.querySelector('[data-testid="pdp-reviews-highlight-banner-host-rating"]');
				if (banner) {
					const ratingDiv = banner.querySelector('div[aria-hidden="true"]');
					const fromBanner = normalize(ratingDiv ? ratingDiv.textContent.trim() : '');
					if (fromBanner) return fromBanner;
				}

				const ratedTextEl = Array.from(document.querySelectorAll('span')).find(el =>
					/Rated\s+[0-9]+(?:\.[0-9]+)?\s+out of 5 stars\./i.test(el.textContent || '')
				);
				const fromRatedText = normalize(ratedTextEl ? ratedTextEl.textContent.trim() : '');
				if (fromRatedText) return fromRatedText;

				const altDiv = document.querySelector('div.rmtgcc3[aria-hidden="true"]');
				const fromAltDiv = normalize(altDiv ? altDiv.textContent.trim() : '');
				if (fromAltDiv) return fromAltDiv;

				return '';
			})()
		`, &rating),

		chromedp.Evaluate(`
			(() => {
				const clean = (txt) => (txt || '').replace(/\s+/g, ' ').trim();

				const preferred = document.querySelector(
					'[data-section-id="DESCRIPTION_DEFAULT"] span.l1h825yc, [data-plugin-in-point-id="DESCRIPTION_DEFAULT"] span.l1h825yc'
				);
				const fromPreferred = clean(preferred ? (preferred.innerText || preferred.textContent) : '');
				if (fromPreferred) return fromPreferred;

				const section = document.querySelector(
					'[data-section-id="DESCRIPTION_DEFAULT"], [data-plugin-in-point-id="DESCRIPTION_DEFAULT"]'
				);
				const fromSection = clean(section ? (section.innerText || section.textContent) : '');
				if (fromSection) return fromSection;

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
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "..."
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
