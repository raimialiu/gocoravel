package concurrent_dictionary

func (d *ConcurrentDictionary[K, V]) TryAdd(key K, value V) bool {
	h := d.getHashCode(key)
	bi := d.bucketIndex(key, &h)
	li := d.lockIndex(bi)

	return d._addInternal(key, value, h, li, bi) != 0
}

func (d *ConcurrentDictionary[K, V]) TryUpdate(key K, value V) bool {
	h := d.getHashCode(key)
	bi := d.bucketIndex(key, &h)
	li := d.lockIndex(bi)

	return d._updateInternal(key, value, h, li, bi) != 0
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

func (d *ConcurrentDictionary[K, V]) Get(key K) *V {
	hashCode := d.getHashCode(key)
	bucketIndex := d.bucketIndex(key, &hashCode)

	return d._getInternal(key, hashCode, bucketIndex)
}

func (d *ConcurrentDictionary[K, V]) TryGet(key K) (bool, *V) {
	value := d.Get(key)
	return value != nil, value
}

func (d *ConcurrentDictionary[K, V]) GetOrDefault(key K, defaultValue V) *V {
	found, value := d.TryGet(key)
	if found {
		return value
	}

	return &defaultValue
}
