package concurrent_dictionary

import (
	"hash/maphash"
	"sync"
)

type (
	Table[K, V any] struct {
		_buckets   []Entry[K, V] // 0-4=> bucket 1
		_locks     []sync.RWMutex
		_lockCount []uint
	}

	Node[T any] struct {
		Value T
	} // will come back to this

	Entry[K, V any] struct {
		Key      K
		Node     *Node[V]
		Next     *Entry[K, V]
		HashCode uint64
	}

	ConcurrentDictionary[K, V any] struct {
		_hashSeed        maphash.Seed
		_concurrentLevel int
		_capacity        int
		_table           *Table[K, V]
	}
)
