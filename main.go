package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"
)

type responses map[string]int

type summary struct {
	identifier int
	times      int
	codes      responses
}

var parallelness = flag.Int("parallel", 8, "How many workers to make")
var runTime = flag.String("runtime", "1s", "Duration to run all workers, eg 1m")

func main() {
	flag.Parse()
	nparallel := *parallelness
	runFor, err := time.ParseDuration(*runTime)
	if err != nil {
		fmt.Printf("Error parsing duration: %s\n", err)
		os.Exit(1)
	}

	stop := make(chan bool)
	summaries := make(chan summary, nparallel)

	for i := 0; i < nparallel; i++ {
		go hammer(i, summaries, stop)
	}

	fmt.Printf("Working for %s...\n", runFor.String())
	time.Sleep(runFor)
	for i := 0; i < nparallel; i++ {
		stop <- true
	}

	var totalTimes int
	responseTotals := make(responses)
	for i := 0; i < nparallel; i++ {
		s := <-summaries
		fmt.Printf("%d did %d iterations. responses: %+v\n", s.identifier, s.times, s.codes)
		totalTimes += s.times
		for status, count := range s.codes {
			responseTotals[status] += count
		}
	}
	fmt.Printf("did %d total runs: %+v\n", totalTimes, responseTotals)
}

func hammer(identifier int, summaries chan summary, stop chan bool) {
	var times int
	s := summary{identifier: identifier}
	s.codes = make(responses)
	for {
		select {
		case <-stop:
			s.times = times
			summaries <- s
			return
		default:
			times++
			client := &http.Client{}
			req, err := http.NewRequest("GET", "http://localhost:8080/metrics", nil)
			if err != nil {
				fmt.Printf("%d: error creating request: %s\n", identifier, err)
				continue
			}
			req.Header.Add("User-Agent", fmt.Sprintf("Plush worker %d", identifier))
			resp, err := client.Do(req)
			if err != nil {
				fmt.Printf("%d: performing response: %s\n", identifier, err)
				continue
			}
			s.codes[resp.Status]++
		}
	}
}
