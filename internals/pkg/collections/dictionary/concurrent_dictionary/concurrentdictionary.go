package concurrent_dictionary

import (
	"fmt"
	"hash/maphash"
	"slices"
	"sync"
)

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

func (d *ConcurrentDictionary[K, V]) TryRemove(key K) bool {
	hashCode := d.getHashCode(key)
	bi := d.bucketIndex(key, &hashCode)
	li := d.lockIndex(bi)

	return d._remove(key, bi, li)
}

func (d *ConcurrentDictionary[K, V]) resize() {
	for i := 0; i < len(d._table._locks)-1; i++ {
		d.acquireLock(i)
	}

	table := CreateNewTable[K, V](NextPrime(d._capacity*2), d._concurrentLevel)

	for i := 0; i < len(d._table._buckets)-1; i++ {
		node := &d._table._buckets[i]

		for node != nil {
			nextNode := node.Next
			newBucketIndex := node.HashCode % uint64(len(table._buckets))
			node.Next = &table._buckets[newBucketIndex]
			table._buckets[newBucketIndex] = *node

			newLockIndex := newBucketIndex % uint64(len(table._locks))
			table._lockCount[newLockIndex]++
			node = nextNode

		}
	}

	d._table = table
	for i := 0; i < len(d._table._locks)-1; i++ {
		d.releaseLock(i)
	}
}
