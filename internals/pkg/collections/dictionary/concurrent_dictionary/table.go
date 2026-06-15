package concurrent_dictionary

import (
	"sync"
)

func CreateNewTable[K, V any](
	capacity int,
	concurrencyLevel int,
) *Table[K, V] {
	buckets := make([]Entry[K, V], capacity)
	locks := make([]sync.RWMutex, concurrencyLevel)
	lockCount := make([]uint, 0)

	return &Table[K, V]{
		_buckets:   buckets,
		_locks:     locks,
		_lockCount: lockCount,
	}
}
