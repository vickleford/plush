package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"
)

func usage() {
	fmt.Fprintf(flag.CommandLine.Output(),
		"Usage: %s [options] http://url.to.hammer/path\nOptions:\n",
		os.Args[0])
	flag.PrintDefaults()
	os.Exit(1)
}

var parallelness = flag.Int("parallel", 1, "How many workers to make")
var runTime = flag.String("duration", "1s", "Duration to run all workers, eg 1m")

type responses map[string]int

type summary struct {
	times  int
	errors int
	codes  responses
}

func main() {
	flag.Usage = usage
	flag.Parse()
	nparallel := *parallelness
	runFor, err := time.ParseDuration(*runTime)
	if err != nil {
		fmt.Printf("Error parsing duration: %s\n", err)
		flag.Usage()
	}
	uri := flag.Arg(0)
	if uri == "" {
		flag.Usage()
	}

	if _, err := makeRequest(uri); err != nil {
		fmt.Printf("Aborting: %s\n", err)
		os.Exit(1)
	}

	stop := make(chan bool)
	summaries := make(chan summary, nparallel)

	fmt.Printf("Starting %d workers for %s...\n", nparallel, runFor.String())
	timebegin := time.Now()
	for i := 0; i < nparallel; i++ {
		go hammer(uri, summaries, stop)
	}

	go func() {
		time.Sleep(runFor)
		for i := 0; i < nparallel; i++ {
			stop <- true
		}
	}()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	go func() {
		<-interrupt
		fmt.Println("shutting down gracefully")
		for i := 0; i < nparallel; i++ {
			stop <- true
		}
	}()

	var totalTimes, totalErrors int
	responseTotals := make(responses)
	for i := 0; i < nparallel; i++ {
		s := <-summaries
		fmt.Printf("%d iterations with %d errors. responses: %+v\n", s.times, s.errors, s.codes)
		totalTimes += s.times
		totalErrors += s.errors
		for status, count := range s.codes {
			responseTotals[status] += count
		}
	}
	fmt.Printf("did %d total runs with %d accumulated errors in %s\n",
		totalTimes,
		totalErrors,
		time.Since(timebegin),
	)
	for status, count := range responseTotals {
		fmt.Printf("\t%d %s\n", count, status)
	}
}

func hammer(uri string, summaries chan summary, stop chan bool) {
	var times, errcount int
	s := summary{}
	s.codes = make(responses)
	for {
		select {
		case <-stop:
			s.times = times
			s.errors = errcount
			summaries <- s
			return
		default:
			times++
			resp, err := makeRequest(uri)
			if err != nil {
				fmt.Printf("Error on request: %s\n", err)
				errcount++
				continue
			}
			s.codes[resp.Status]++
		}
	}
}

func makeRequest(uri string) (*http.Response, error) {
	client := &http.Client{}
	defer client.CloseIdleConnections()
	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("User-Agent", "Plush")
	resp, err := client.Do(req)
	if err != nil {
		return resp, err
	}
	defer resp.Body.Close()
	return resp, err
}
