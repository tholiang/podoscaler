package main

import (
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net/http"
	"time"
	"context"
	"sync"
	"io"
	"runtime"
	// "github.com/prometheus/client_golang/prometheus"
)

var (
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
)

// var httpResponseTime = prometheus.NewHistogramVec(
// 	prometheus.HistogramOpts{
// 		Name:    "http_response_time_milliseconds",
// 		Help:    "Response time in milliseconds",
// 		Buckets: []float64{0.0001, 0.001, 0.01, 0.1, 1, 2, 3, 4, 5, 10, 15, 25, 40, 50, 100, 250, 500, 1000, 2500, 5000},
// 	},
// 	[]string{"path"}, // Label responses by endpoint
// )

func index(w http.ResponseWriter, r *http.Request) {
	// start := time.Now()
	fmt.Fprintf(w, "<h1>hello</h1>")
	// duration := time.Since(start).Seconds() * 1000 // ms
	// httpResponseTime.WithLabelValues(r.URL.Path).Observe(duration)
}

func setThreads(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var numThreads int
	b, err := io.ReadAll(r.Body)
	if err != nil {
		fmt.Fprint(w, err)
		return
	}
	err = json.Unmarshal(b, &numThreads)
	if err != nil {
		fmt.Fprint(w, err)
		return
	}

	// Cancel previous goroutines
	cancel()
	wg.Wait()

	// Start new context and goroutines
	ctx, cancel = context.WithCancel(context.Background())

	for i := 0; i < numThreads; i++ {
		wg.Add(1)
		go wasteCPU(ctx)
		time.Sleep(100 * time.Millisecond)
	}

	fmt.Fprintf(w, "Set active thread count to %d!", runtime.NumGoroutine())
}

func wasteCPU(ctx context.Context) {
	defer wg.Done()
	sum := 0
	for {
		select {
		case <-ctx.Done():
			return
		default:
			sum += rand.IntN(100)
		}
	}
}

func main() {
	// reg := prometheus.NewRegistry()

	// // Add go runtime metrics, process collectors, and http response time
	// reg.MustRegister(
	// 	collectors.NewGoCollector(),
	// 	collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	// 	httpResponseTime,
	// )

	// // handle prometheus scraping
	// http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{Registry: reg}))
	ctx, cancel = context.WithCancel(context.Background())

	http.HandleFunc("/", index)
	http.HandleFunc("/setthreads", setThreads)

	// log.Println("testapp server running on :3000")
	http.ListenAndServe(":3000", nil)
	// for {
	// 	go wasteCPU()
	// 	time.Sleep(100 * time.Millisecond)
	// }
}
