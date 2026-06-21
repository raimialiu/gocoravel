package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/raimialiu/gocoravel/internals/pkg/cronna"
)

func main() {
	//fmt.Println("Testing concurrent dictionary")
	/*
		dict := concurrent_dictionary.New[int, string](nil, nil)
		dict.TryAdd(2, "Taye")
		dict.TryAdd(3, "Kehinde")
		dict.TryAdd(4, "another one")
		dict.TryAdd(2, "bimbo")

		fmt.Println(*dict.Get(4))
		deleted := dict.TryRemove(3)
		fmt.Println(deleted)
		fmt.Println(*dict.Get(3))
	*/
	c := cronna.New()
	err := c.Start()
	if err != nil {
		return
	}
	cronExpression := "*/1 * * * *"
	c.AddFunc(cronExpression, func() error {
		fmt.Println(time.Now())
		fmt.Println("testing my cronna for running cronna")
		return nil
	})

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	c.Stop()

	// c.AddFunc("*/2 * * * *", func(ctx context.Context) error {
	// fmt.Println(time.Now())
	//fmt.Println("testing my cronna for running crons")
	//return nil
	//})

	// For core functionalities, I am supporting
	// V1
	//	- Schedule (scheduler by in the custom interface) -> 1
	//	- Event -> 2
	//	- Message Distributor (rudderstack) - 4
	//	- Queue -> 3
	//	- Redis store only
	//  - C# Delegate Pattern - 5
	// V2 (multiple store configurable)
}
