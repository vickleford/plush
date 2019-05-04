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
	times int
	codes responses
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
	uri := flag.Arg(0)
	if uri == "" {
		fmt.Printf("give a url at the end")
		os.Exit(1)
	}

	if _, err := makeRequest(uri); err != nil {
		fmt.Printf("Aborting: %s\n", err)
		os.Exit(1)
	}

	stop := make(chan bool)
	summaries := make(chan summary, nparallel)

	for i := 0; i < nparallel; i++ {
		go hammer(uri, summaries, stop)
	}

	fmt.Printf("Starting %d workers for %s...\n", nparallel, runFor.String())
	time.Sleep(runFor)
	for i := 0; i < nparallel; i++ {
		stop <- true
	}

	var totalTimes int
	responseTotals := make(responses)
	for i := 0; i < nparallel; i++ {
		s := <-summaries
		fmt.Printf("%d iterations. responses: %+v\n", s.times, s.codes)
		totalTimes += s.times
		for status, count := range s.codes {
			responseTotals[status] += count
		}
	}
	fmt.Printf("did %d total runs: %+v\n", totalTimes, responseTotals)
}

func hammer(uri string, summaries chan summary, stop chan bool) {
	var times int
	s := summary{}
	s.codes = make(responses)
	for {
		select {
		case <-stop:
			s.times = times
			summaries <- s
			return
		default:
			times++
			resp, err := makeRequest(uri)
			if err != nil {
				fmt.Printf("Error on request: %s\n", err)
				continue
			}
			s.codes[resp.Status]++
		}
	}
}

func makeRequest(uri string) (*http.Response, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("User-Agent", "Plush")
	resp, err := client.Do(req)
	if err != nil {
		return resp, err
	}
	return resp, err
}
