package concurrent_dictionary

import (
	"fmt"
	"hash/maphash"
	"slices"
	"sync"
)

func NewConcurrentDictionary[K, V any](
	concurrentLevel *int,
	capacity *int,
) *ConcurrentDictionary[K, V] {
	maxCapacity := SetCapacity(capacity)
	numberOfLocks := SetConcurrencyLevel(concurrentLevel)

	var fixedSeed = maphash.MakeSeed()

	if !IsPrime(maxCapacity) {
		panic("Max capacity not prime")
	}

	if numberOfLocks > maxCapacity {
		panic("too many locks")
	}

	return &ConcurrentDictionary[K, V]{
		_hashSeed:        fixedSeed,
		_concurrentLevel: numberOfLocks,
		_capacity:        maxCapacity,
		_table:           CreateNewTable[K, V](maxCapacity, numberOfLocks),
	}
}

func (d *ConcurrentDictionary[K, V]) getHashCode(key K) uint64 {
	var h maphash.Hash
	h.SetSeed(d._hashSeed)

	switch v := any(key).(type) {
	case string:
		_, _ = h.WriteString(v)
	case []byte:
		_, _ = h.Write(v)
	case fmt.Stringer:
		_, _ = h.WriteString(v.String())
	default:
		stringValue := fmt.Sprintf("%v", v)
		_, _ = h.WriteString(stringValue)
	}

	return h.Sum64() & 0x7FFFFFFF
}

func (d *ConcurrentDictionary[K, V]) bucketIndex(key K, hashCode *uint64) int {
	hashKey := uint64(0)
	if hashCode == nil {
		hashKey = d.getHashCode(key)
	} else {
		hashKey = *hashCode
	}

	bucketIndex := int(hashKey) % len(d._table._buckets)
	return bucketIndex
}

func (d *ConcurrentDictionary[K, V]) lockIndex(bucketIndex int) int {
	lockIndex := bucketIndex % len(d._table._locks)
	return lockIndex
}

func (d *ConcurrentDictionary[K, V]) createLockAt(lockIndex int) *sync.RWMutex {
	if &d._table._locks[lockIndex] != nil {
		return &d._table._locks[lockIndex]
	}
	slices.Insert(d._table._locks, lockIndex, sync.RWMutex{})
	return &d._table._locks[lockIndex]
}

func (d *ConcurrentDictionary[K, V]) acquireLock(lockIndex int) {
	d.createLockAt(lockIndex)
	d._table._locks[lockIndex].Lock()
}

func (d *ConcurrentDictionary[K, V]) releaseLock(lockIndex int) {
	d._table._locks[lockIndex].Unlock()
}

func (d *ConcurrentDictionary[K, V]) TryGet(key K) (bool, *V) {
	value := d.Get(key)
	return value != nil, value
}

func (d *ConcurrentDictionary[K, V]) LoadFactor(lockIndex int) *V {
	d._table._lockCount[lockIndex]
}

func (d *ConcurrentDictionary[K, V]) Get(key K) *V {
	hashCode := d.getHashCode(key)
	bucketIndex := d.bucketIndex(key, &hashCode)

	node := &d._table._buckets[bucketIndex]
	for {
		if node == nil || (node != nil && node.Node == nil) {
			return nil
		}

		if node.HashCode == hashCode {
			if ValueOf(node.Key) == ValueOf(key) {
				return &node.Node.Value
			}
		}

		node = node.Next
	}
}

func (d *ConcurrentDictionary[K, V]) TryAdd(key K, value V) bool {
	hashCode := d.getHashCode(key)
	buckIndex := d.bucketIndex(key, &hashCode)
	lockIndex := d.lockIndex(buckIndex)

	d.acquireLock(lockIndex)
	defer d.releaseLock(lockIndex)
	keyValue := d.Get(key)
	if keyValue != nil {
		return false
	}

	oldEntry := &d._table._buckets[buckIndex]
	d._table._buckets[buckIndex] = *NewEntry(key, value, hashCode, oldEntry)
	d._table._lockCount[lockIndex]++
	return true
}
