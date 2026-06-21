package concurrent_dictionary

import "runtime"

func DefaultConfig() DictionaryOptions {
	return DictionaryOptions{
		concurrency: runtime.NumCPU(),
		capacity:    DEFAULT_CAPACITY, // smallest prime ≥ 31
		loadFactor:  LOAD_FACTOR,
	}
}

// WithConcurrency sets the number of lock stripes. Higher values reduce
// contention at the cost of memory. Must be at least 1.
func WithConcurrencyLevel(n int) DictionaryOpt {
	return func(c *DictionaryOptions) {
		if n >= 1 {
			c.concurrency = n
		}
	}
}

// WithCapacity sets the initial bucket count. It is rounded up to the next
// prime internally.
func WithCapacity(n int) DictionaryOpt {
	return func(c *DictionaryOptions) {
		if n >= 1 {
			c.capacity = n
		}
	}
}

// WithLoadFactor sets the threshold at which the map triggers a resize.
// Typical values are 0.5–0.9.
func WithLoadFactor(f float64) DictionaryOpt {
	return func(c *DictionaryOptions) {
		if f > 0 {
			c.loadFactor = f
		}
	}
}
