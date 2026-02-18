package utils

import (
	"math/rand"
	"time"
)

// RandomDelay sleeps for a random duration between min and max.
// Pass time.Duration values like: RandomDelay(2*time.Second, 5*time.Second)
//
// WHY RANDOM? Fixed delays are detectable patterns.
// Random delays look more like a human browsing.
func RandomDelay(min, max time.Duration) {
	diff := max - min
	sleep := min + time.Duration(rand.Int63n(int64(diff)))
	time.Sleep(sleep)
}