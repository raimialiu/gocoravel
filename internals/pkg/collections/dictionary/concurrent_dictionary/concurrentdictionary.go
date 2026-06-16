package concurrent_dictionary

import (
	"hash/maphash"
	"reflect"
	"runtime"
	"slices"
	"sync"
)

func setCapacity(count *int) int {
	capacity := 0
	if count == nil {
		capacity = DEFAULT_CAPACITY
	} else {
		capacity = *count
	}

	return capacity
}

func setConcurrencyLevel(level *int) int {
	concurrencyLevel := 0
	if level != nil {
		concurrencyLevel = *level
	} else {
		concurrencyLevel = runtime.NumCPU()
	}

	return concurrencyLevel
}

func NewConcurrentDictionary[K, V any](
	concurrentLevel *int,
	capacity *int,
) *ConcurrentDictionary[K, V] {
	maxCapacity := setCapacity(capacity)
	numberOfLocks := setConcurrencyLevel(concurrentLevel)

	var fixedSeed = maphash.MakeSeed()

	return &ConcurrentDictionary[K, V]{
		_hashSeed:        fixedSeed,
		_concurrentLevel: numberOfLocks,
		_capacity:        maxCapacity,
		_table:           CreateNewTable[K, V](maxCapacity, numberOfLocks),
	}
}

func (d *ConcurrentDictionary[K, V]) getHashCode(key K) uint64 {
	var keyReflect = reflect.ValueOf(key)
	actualValue := keyReflect.Interface()

	hashKey := actualValue.(string)
	var h maphash.Hash
	h.SetSeed(d._hashSeed)
	_, err := h.WriteString(hashKey)
	if err != nil {
		panic(err)
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
	slices.Insert(d._table._locks, lockIndex, sync.RWMutex{})
	return &d._table._locks[lockIndex]
}

func (d *ConcurrentDictionary[K, V]) acquireLock(lockIndex int) {
	d._table._locks[lockIndex].Lock()
}

func (d *ConcurrentDictionary[K, V]) Get(key K) *V {
	hashCode := new(d.getHashCode(key))
	bucketIndex := d.bucketIndex(key, hashCode)
	lockIndex := d.lockIndex(bucketIndex)

	lock := &d._table._locks[lockIndex]
	if lock == nil {
		lock = d.createLockAt(bucketIndex)
	}

	node := &d._table._buckets[bucketIndex]
	if node == nil || (node != nil && node.Node == nil) {
		return nil
	}

	for {
		if node.HashCode == *hashCode {
			return &node.Node.Value
		}

	}
}

func (d *ConcurrentDictionary[K, V]) TryAdd(key K, value V) bool {
	hashCode := new(d.getHashCode(ke))
	buckIndex := d.bucketIndex(key, hashCode)
	lockIndex := d.lockIndex(buckIndex)

	d.acquireLock(lockIndex)
	keyValue := d.Get(key)
	if keyValue != nil {
		return false
	}

	oldEntry := &d._table._buckets[buckIndex]
	d._table._buckets[buckIndex] = *NewEntry(key, value, *hashCode, oldEntry)
	return true
}
