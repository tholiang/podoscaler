package main

import (
	"fmt"
	"math/rand/v2"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var httpResponseTime = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "http_response_time_milliseconds",
		Help:    "Response time in milliseconds",
		Buckets: []float64{0.0001, 0.001, 0.01, 0.1, 1, 2, 3, 4, 5, 10, 15, 25, 40, 50, 100, 250, 500, 1000, 2500, 5000},
	},
	[]string{"path"}, // Label responses by endpoint
)

func index(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	fmt.Fprintf(w, "<h1>hello</h1>")
	duration := time.Since(start).Seconds() * 1000 // ms
	httpResponseTime.WithLabelValues(r.URL.Path).Observe(duration)
}

func noop(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	b := make([]byte, r.ContentLength)
	r.Body.Read(b)
	w.Write(b)
	duration := time.Since(start).Seconds() * 1000 // ms
	httpResponseTime.WithLabelValues(r.URL.Path).Observe(duration)
}

func wasteCPU() {
	randomsum := 0
	for {
		randomsum += rand.IntN(100)
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

	// http.HandleFunc("/", index)
	// http.HandleFunc("/noop", noop)

	// log.Println("testapp server running on :3000")
	// http.ListenAndServe(":3000", nil)
	for {
		go wasteCPU()
		time.Sleep(100 * time.Millisecond)
	}
}
