package concurrent_dictionary

import (
	"hash/maphash"
	"sync"
)

func New[K, V any](options ...DictionaryOpt) *ConcurrentDictionary[K, V] {
	opts := DefaultConfig()
	for _, opt := range options {
		opt(&opts)
	}

	if opts.concurrency > opts.concurrency {
		panic("too many locks")
	}

	t := CreateNewTable[K, V](NextPrime(opts.capacity), opts.concurrency)
	buckets := make([]Entry[K, V], opts.capacity)
	t._buckets = buckets
	t._locks = make([]sync.RWMutex, opts.concurrency)
	t._lockCount = make([]uint, opts.concurrency)

	for i := 0; i < opts.concurrency-1; i++ {
		t._locks[i] = sync.RWMutex{}
	}

	return &ConcurrentDictionary[K, V]{
		_table:           t,
		_capacity:        opts.capacity,
		_concurrentLevel: opts.concurrency,
		_hashSeed:        maphash.MakeSeed(),
		_loadFactor:      opts.loadFactor,
	}
}
