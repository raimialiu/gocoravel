package concurrent_dictionary

func (d ConcurrentDictionary[K, V]) _getInternal(key K, hashCode uint64, bucketIndex int) *V {
	node := &d._table._buckets[bucketIndex]

	var zero K
	for node != nil {
		if node.HashCode == 0 && ValueOf(node.Key) == ValueOf(zero) {
			break
		}

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

	result := false

	// 42
	// 42 = 3 -> 4 -> 2 -> nil
	// 42 = 3 -> nil
	// 42 = nil
	previous := new(Entry[K, V]) // 3
	for node != nil {            // 4
		if node.HashCode == h && ValueOf(node.Key) == ValueOf(key) { // 4
			nextNode := node.Next // 2
			if previous != nil && previous.Node != nil {
				previous.Next = nextNode
				result = true
				break
			}

			// 4
			if nextNode != nil && nextNode.Node != nil { // 3
				node = nextNode
				d._table._buckets[bi] = *node
				result = true
				break
			} else {
				d._table._buckets[bi] = *new(Entry[K, V])
				result = true
				break
			}

		}

		previous = node  // 4
		node = node.Next // 2
	}

	return result
}
