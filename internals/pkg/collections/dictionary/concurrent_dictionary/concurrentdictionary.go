package concurrent_dictionary

import (
	"hash/maphash"
	"reflect"
	"runtime"
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

func (d *ConcurrentDictionary[K, V]) bucketIndex(key K) int {
	hashCode := d.getHashCode(key)
	bucketIndex := int(hashCode) % len(d._table._buckets)

	return bucketIndex
}

func (d *ConcurrentDictionary[K, V]) lockIndex(bucketIndex int) int {
	lockIndex := bucketIndex % len(d._table._locks)
	return lockIndex
}

func (d *ConcurrentDictionary[K, V]) Get(key K) V {
	hashCode := d.getHashCode(key)
	bucketIndex := d.bucketIndex(key)

	node := d._table._buckets[bucketIndex]

	for {
		
	}
}

func (d *ConcurrentDictionary[K, V]) Add(key K, value V) {

}
