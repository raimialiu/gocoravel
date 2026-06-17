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

func (d *ConcurrentDictionary[K, V]) TryGet(key K) (bool, *V) {
	value := d.Get(key)
	return value != nil, value
}

func (d *ConcurrentDictionary[K, V]) Get(key K) *V {
	hashCode := d.getHashCode(key)
	bucketIndex := d.bucketIndex(key, &hashCode)

	return d._getInternal(key, hashCode, bucketIndex)
}

func (d *ConcurrentDictionary[K, V]) GetOrDefault(key K, defaultValue V) *V {
	found, value := d.TryGet(key)
	if found {
		return value
	}

	return &defaultValue
}

func (d *ConcurrentDictionary[K, V]) TryRemove(key K) bool {
	hashCode := d.getHashCode(key)
	bi := d.bucketIndex(key, &hashCode)
	li := d.lockIndex(bi)

	return d._remove(key, bi, li)
}

func (d *ConcurrentDictionary[K, V]) Count() int {
	count := 0
	for i := 0; i < len(d._table._locks); i++ {
		d.acquireLock(i)
	}

	for i := 0; i < len(d._table._lockCount); i++ {
		current := d._table._lockCount[i]
		count += int(current)
	}

	for i := 0; i < len(d._table._locks); i++ {
		d.releaseLock(i)
	}

	return count
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

	needFactorIndex := (float64(len(d._table._buckets))) / (float64(d._concurrentLevel) * d._loadFactor)
	if float64(d._table._lockCount[lockIndex]) > needFactorIndex {
		d.resize()
	}

	return true
}
