package concurrent_dictionary

import (
	"hash/maphash"
	"sync"

	"github.com/raimialiu/gostream/stream"
)

func New[K, V any](options ...DictionaryOpt) *ConcurrentDictionary[K, V] {
	opts := DefaultConfig()

	optionList := stream.From(options).
		Filter(func(opt DictionaryOpt) bool { return opt != nil }).
		ToList()

	for _, opt := range optionList {
		if opt != nil {
			opt(&opts)
		}
	}

	if opts.concurrency > opts.capacity {
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
