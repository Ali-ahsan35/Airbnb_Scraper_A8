package utils

import (
	"context"
	"math/rand"

	"github.com/chromedp/chromedp"
)

// userAgents — real browser strings we rotate through each session.
// Airbnb checks User-Agent to detect bots. By rotating these,
// each scrape session looks like a different real browser.
var userAgents = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36",
}

func RandomUserAgent() string {
	return userAgents[rand.Intn(len(userAgents))]
}

// StealthOpts returns ChromeDP browser launch options that hide automation.
//
// Key flags:
//   - disable-blink-features=AutomationControlled → removes navigator.webdriver flag
//   - headless=new → uses Chrome's newer headless mode (harder to detect)
//   - WindowSize → bots often have tiny/default windows; we set a normal size
func StealthOpts(headless bool) []chromedp.ExecAllocatorOption {
	opts := []chromedp.ExecAllocatorOption{
		chromedp.NoFirstRun,
		chromedp.NoDefaultBrowserCheck,
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
		chromedp.Flag("excludeSwitches", "enable-automation"),
		chromedp.Flag("useAutomationExtension", false),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.WindowSize(1920, 1080),
		chromedp.UserAgent(RandomUserAgent()),
	}

	if headless {
		opts = append(opts, chromedp.Flag("headless", "new"))
	}

	return opts
}

// HideWebDriver injects JavaScript into the page to remove
// telltale signs of automation that Airbnb's scripts look for.
//
// Even with the flags above, some sites run JS checks on the page itself.
// This patches those JS properties before any scraping happens.
func HideWebDriver() chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		return chromedp.Evaluate(`
			Object.defineProperty(navigator, 'webdriver', { get: () => undefined });
			Object.defineProperty(navigator, 'plugins', { get: () => [1, 2, 3, 4, 5] });
			Object.defineProperty(navigator, 'languages', { get: () => ['en-US', 'en'] });
		`, nil).Do(ctx)
	})
}