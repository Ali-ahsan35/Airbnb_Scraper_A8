package utils

import (
	"fmt"
	"time"
)

// Retry runs fn up to maxRetries times.
// If fn returns nil (success) it stops immediately.
// If fn keeps failing, it waits longer each attempt (exponential backoff)
// and returns the last error after all attempts are exhausted.
//
// EXPONENTIAL BACKOFF means:
//   attempt 1 fails → wait 2 seconds
//   attempt 2 fails → wait 4 seconds
//   attempt 3 fails → wait 8 seconds
//
// WHY? If Airbnb is rate-limiting you, hammering it again immediately
// makes it worse. Waiting longer each time gives it time to settle.
//
// Usage:
//
//	err := utils.Retry(3, func() error {
//	    return scraper.ScrapePage(url)
//	})
func Retry(maxRetries int, fn func() error) error {
	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		lastErr = fn()
		if lastErr == nil {
			return nil // success — stop retrying
		}

		if attempt < maxRetries {
			wait := time.Duration(1<<uint(attempt)) * time.Second // 2s, 4s, 8s...
			Warn("Attempt %d/%d failed: %v — retrying in %v", attempt, maxRetries, lastErr, wait)
			time.Sleep(wait)
		}
	}

	return fmt.Errorf("all %d attempts failed — last error: %w", maxRetries, lastErr)
}