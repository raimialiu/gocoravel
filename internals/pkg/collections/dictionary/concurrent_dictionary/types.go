package concurrent_dictionary

import (
	"hash/maphash"
	"sync"
)

// -----------------------------------------------------------------------------
// Types
// -----------------------------------------------------------------------------
type (

	// table is the internal shard structure: buckets, per-shard locks, and item
	// counts per lock stripe.
	Table[K, V any] struct {
		_buckets   []Entry[K, V] // 0-4=> bucket 1
		_locks     []sync.RWMutex
		_lockCount []uint
	}

	Node[T any] struct {
		Value T
	} // will come back to this

	// entry is a singly-linked node inside a hash bucket.
	Entry[K, V any] struct {
		Key      K
		Node     *Node[V]
		Next     *Entry[K, V]
		HashCode uint64
	}

	DictionaryOptions struct {
		concurrency int
		capacity    int
		loadFactor  float64
	}

	// Option configures a Map at construction time.
	DictionaryOpt func(*DictionaryOptions)

	ConcurrentDictionary[K, V any] struct {
		_hashSeed        maphash.Seed
		_concurrentLevel int
		_capacity        int
		_table           *Table[K, V]
		_loadFactor      float64
	}
)
