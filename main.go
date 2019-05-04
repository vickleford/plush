package main

import (
	"fmt"
	"sync"
	"time"
)

type summary struct {
	identifier int
	took       time.Duration
	message    string
}

func main() {
	nparallel := 3
	stop := make(chan bool)
	summaries := make(chan summary)

	for i := 0; i < nparallel; i++ {
		fmt.Printf("starting id %d\n", i)
		go hammer(i, summaries, stop)
	}

	var wg sync.WaitGroup
	wg.Add(nparallel)
	for i := 0; i < nparallel; i++ {
		go report(summaries, &wg)
	}

	fmt.Println("waiting until defined time...")
	time.Sleep(1 * time.Second)
	fmt.Println("shutting them all down now.")
	for i := 0; i < nparallel; i++ {
		stop <- true
	}
	// great. now you need to wait for all the report goroutines to finish.
	wg.Wait()
	fmt.Println("exiting main")
}

func hammer(identifier int, summaries chan summary, stop chan bool) {
	start := time.Now()
	var times int
	for {
		select {
		case <-stop:
			message := fmt.Sprintf("did %d iterations", times)
			s := summary{
				identifier: identifier,
				took:       time.Since(start),
				message:    message,
			}
			fmt.Printf("%d returning\n", identifier)
			summaries <- s
			return
		default:
			times++
		}
	}
}

func report(s chan summary, wg *sync.WaitGroup) {
	defer wg.Done()
	fmt.Println("waiting for summary")
	smry := <-s
	fmt.Printf("%d %s\n", smry.identifier, smry.message)
}
