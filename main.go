package main

import (
	"fmt"

	"github.com/raimialiu/gocoravel/internals/pkg/collections/dictionary/concurrent_dictionary"
)

func main() {
	fmt.Println("Testing concurrent dictionary")
	dict := concurrent_dictionary.NewConcurrentDictionary[int, string](nil, nil)
	dict.TryAdd(2, "Taye")
	dict.TryAdd(3, "Kehinde")
	dict.TryAdd(4, "another one")
	dict.TryAdd(2, "bimbo")

	fmt.Println(*dict.Get(4))
}
