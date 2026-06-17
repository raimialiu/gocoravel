package concurrent_dictionary

func (d ConcurrentDictionary[K, V]) _getInternal(key K, hashCode uint64, bucketIndex int) *V {
	node := &d._table._buckets[bucketIndex]

	for node != nil {
		if node.HashCode == hashCode {
			if ValueOf(node.Key) == ValueOf(key) {
				return &node.Node.Value
			}
		}
		node = node.Next
	}

	return nil
}

func (d ConcurrentDictionary[K, V]) _remove(key K, bi, li int) bool {
	d.acquireLock(li)
	defer d.releaseLock(li)
	node := &d._table._buckets[bi]
	h := d.getHashCode(key)

	// 42
	// 42 = 3 -> 4 -> 2 -> nil
	// 42 = 3 -> nil
	// 42 = nil
	previous := node  // 3
	for node != nil { // 4
		if node.HashCode == h && ValueOf(node.Key) == ValueOf(key) { // 4
			nextNode := node.Next // 2
			if previous != nil {
				previous.Next = nextNode
				return true
			}

			// 4
			if nextNode != nil { // 3
				node = nextNode
				d._table._buckets[bi] = *node
			} else {
				node = nil
				d._table._buckets[bi] = *node
			}

		}

		previous = node  // 4
		node = node.Next // 2
	}

	return false
}
