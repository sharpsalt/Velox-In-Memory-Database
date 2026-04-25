package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

/*
velox-benchmark
A high-performance concurrent benchmark tool for the Velox Database.

Simulates multiple concurrent clients sending requests as fast as possible.
Reports throughput (requests/sec) and latency percentiles.

Usage:
  velox-benchmark -c 50 -n 100000 -t PING,SET,GET
*/

var (
	host        = flag.String("h", "127.0.0.1", "Server hostname")
	port        = flag.Int("p", 7379, "Server port")
	clients     = flag.Int("c", 50, "Number of parallel connections")
	requests    = flag.Int("n", 100000, "Total number of requests")
	tests       = flag.String("t", "PING,SET,GET", "Comma separated list of tests to run")
	payloadSize = flag.Int("d", 3, "Data size of SET/GET value in bytes")
	pipeline    = flag.Int("q", 1, "Pipeline <numreq> requests. Default 1 (no pipeline).")
)

type TestStats struct {
	Name       string
	Requests   int
	Errors     int
	Duration   time.Duration
	Throughput float64
}

func main() {
	flag.Parse()

	addr := fmt.Sprintf("%s:%d", *host, *port)
	fmt.Printf("====== Velox Benchmark ======\n")
	fmt.Printf("Server: %s\n", addr)
	fmt.Printf("Clients: %d\n", *clients)
	fmt.Printf("Requests per test: %d\n", *requests)
	fmt.Printf("Payload size: %d bytes\n\n", *payloadSize)

	// Check if server is reachable
	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		log.Fatalf("Could not connect to server at %s: %v", addr, err)
	}
	conn.Close()

	testList := strings.Split(*tests, ",")
	for _, test := range testList {
		runTest(strings.TrimSpace(test), addr)
	}
}

func runTest(testName, addr string) {
	fmt.Printf("Running %s test...\n", testName)

	var payload string
	for i := 0; i < *payloadSize; i++ {
		payload += "x"
	}

	// Commands pre-formatted as RESP to avoid overhead during benchmark
	var cmdData []byte
	switch strings.ToUpper(testName) {
	case "PING":
		cmdData = []byte("*1\r\n$4\r\nPING\r\n")
	case "SET":
		cmdData = []byte(fmt.Sprintf("*3\r\n$3\r\nSET\r\n$3\r\nfoo\r\n$%d\r\n%s\r\n", *payloadSize, payload))
	case "GET":
		cmdData = []byte("*2\r\n$3\r\nGET\r\n$3\r\nfoo\r\n")
	case "INCR":
		cmdData = []byte("*2\r\n$4\r\nINCR\r\n$7\r\ncounter\r\n")
	default:
		fmt.Printf("Unknown test: %s. Skipping.\n", testName)
		return
	}

	var pipelineReqs = *pipeline
	if pipelineReqs < 1 {
		pipelineReqs = 1
	}

	// Create pipelined payload
	pipelinedData := make([]byte, 0, len(cmdData)*pipelineReqs)
	for i := 0; i < pipelineReqs; i++ {
		pipelinedData = append(pipelinedData, cmdData...)
	}

	var wg sync.WaitGroup
	var completedReqs uint64
	var errorCount uint64

	reqsPerClient := *requests / *clients
	batchesPerClient := reqsPerClient / pipelineReqs
	if batchesPerClient == 0 {
		batchesPerClient = 1
	}

	startTime := time.Now()

	for i := 0; i < *clients; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			conn, err := net.Dial("tcp", addr)
			if err != nil {
				atomic.AddUint64(&errorCount, uint64(reqsPerClient))
				return
			}
			defer conn.Close()
			reader := bufio.NewReader(conn)

			for j := 0; j < batchesPerClient; j++ {
				_, err := conn.Write(pipelinedData)
				if err != nil {
					atomic.AddUint64(&errorCount, uint64(pipelineReqs))
					continue
				}

				for p := 0; p < pipelineReqs; p++ {
					// Read the first line of the response
					resp, err := reader.ReadString('\n')
					if err != nil {
						atomic.AddUint64(&errorCount, 1)
						continue
					}
					
					// If it's a bulk string (like from GET), we MUST read the second line
					if len(resp) > 0 && resp[0] == '$' && resp != "$-1\r\n" {
						_, err = reader.ReadString('\n')
						if err != nil {
							atomic.AddUint64(&errorCount, 1)
							continue
						}
					}

					atomic.AddUint64(&completedReqs, 1)
				}
			}
		}()
	}

	wg.Wait()
	duration := time.Since(startTime)
	throughput := float64(completedReqs) / duration.Seconds()

	fmt.Printf("  %d requests completed in %.2f seconds\n", completedReqs, duration.Seconds())
	fmt.Printf("  %d parallel clients\n", *clients)
	if errorCount > 0 {
		fmt.Printf("  %d errors\n", errorCount)
	}
	fmt.Printf("  %.2f requests per second\n\n", throughput)
}
