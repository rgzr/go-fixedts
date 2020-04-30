package main

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/rgzr/go-fixedts"
)

var desiredTimestep float64 = 1.0 / 60.0 // 60 steps per second (of 33 ms each one aprox)

func main() {
	wg := &sync.WaitGroup{}
	ctx, cancelFunc := context.WithCancel(context.Background())
	wg.Add(2)

	slowTS := fixedts.NewWithConfig(&fixedts.FixedTimestepConfig{
		DesiredTimestep: desiredTimestep,
		Context:         ctx,
		WaitGroup:       wg,
	}, func(ts *fixedts.FixedTimestep) {
		log.Println("Slow working!")
		time.Sleep(time.Millisecond * 30)
	})

	fastTS := fixedts.NewWithConfig(&fixedts.FixedTimestepConfig{
		DesiredTimestep: desiredTimestep,
		Context:         ctx,
		WaitGroup:       wg,
	}, func(ts *fixedts.FixedTimestep) {
		log.Println("Fast working!")
		time.Sleep(time.Millisecond * 5)
	})

	// Both working
	time.Sleep(time.Second * 1)

	// Both paused
	slowTS.Update(&fixedts.FixedTimestepConfig{
		Paused:          true,
		DesiredTimestep: desiredTimestep,
		Context:         ctx,
		WaitGroup:       wg,
	})

	fastTS.Update(&fixedts.FixedTimestepConfig{
		Paused:          true,
		DesiredTimestep: desiredTimestep,
		Context:         ctx,
		WaitGroup:       wg,
	})

	time.Sleep(time.Second * 1)

	// Slow paused
	slowTS.Update(&fixedts.FixedTimestepConfig{
		Paused:          false,
		DesiredTimestep: desiredTimestep,
		Context:         ctx,
		WaitGroup:       wg,
	})

	time.Sleep(time.Second * 1)

	// Fast paused
	slowTS.Update(&fixedts.FixedTimestepConfig{
		Paused:          true,
		DesiredTimestep: desiredTimestep,
		Context:         ctx,
		WaitGroup:       wg,
	})

	fastTS.Update(&fixedts.FixedTimestepConfig{
		Paused:          false,
		DesiredTimestep: desiredTimestep,
		Context:         ctx,
		WaitGroup:       wg,
	})

	time.Sleep(time.Second * 1)

	// Play 5 steps
	fastTS.Update(&fixedts.FixedTimestepConfig{
		Paused:          true,
		DesiredTimestep: desiredTimestep,
		Context:         ctx,
		WaitGroup:       wg,
	})

	slowTS.Update(&fixedts.FixedTimestepConfig{
		Paused:          false,
		DesiredTimestep: desiredTimestep,
		StepsUntilPause: 5,
		Context:         ctx,
		WaitGroup:       wg,
	})

	// Cancel
	cancelFunc()
	wg.Wait()
	log.Println("Bye!")
}
