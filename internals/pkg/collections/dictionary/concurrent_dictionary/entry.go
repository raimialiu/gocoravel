package concurrent_dictionary

func NewNode[T any](value T) *Node[T] {
	return &Node[T]{
		Value: value,
	}
}

func NewEntry[K, V any](key K, value V, hashCode uint64, next *Entry[K, V]) *Entry[K, V] {
	return &Entry[K, V]{
		Key:      key,
		Node:     NewNode(value),
		Next:     next,
		HashCode: hashCode,
	}
}
