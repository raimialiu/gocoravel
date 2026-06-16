package concurrent_dictionary

import (
	"math"
	"reflect"
	"runtime"
)

func ValueOf[K any](data K) interface{} {
	var keyReflect = reflect.ValueOf(data)
	actualValue := keyReflect.Interface()

	return actualValue.(interface{})
}

func SetCapacity(count *int) int {
	capacity := 0
	if count == nil {
		capacity = DEFAULT_CAPACITY
	} else {
		capacity = *count
	}

	return capacity
}

func SetConcurrencyLevel(level *int) int {
	concurrencyLevel := 0
	if level != nil {
		concurrencyLevel = *level
	} else {
		concurrencyLevel = runtime.NumCPU()
	}

	return concurrencyLevel
}

func IsPrime(n int) bool {
	// Numbers less than 2 are not prime
	if n < 2 {
		return false
	}

	// 2 is the only even prime number
	if n == 2 {
		return true
	}

	// Eliminate all other even numbers
	if n%2 == 0 {
		return false
	}

	// Check odd divisors up to √n
	// No need to go beyond √n because if n = a * b,
	// then one of a or b must be ≤ √n
	limit := int(math.Sqrt(float64(n)))
	for i := 3; i <= limit; i += 2 {
		if n%i == 0 {
			return false
		}
	}

	return true
}
