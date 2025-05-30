package main

import (
	"fmt"
	"log"
	"net/http"
	"time"
	"math/rand"
)

func index(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "<h1>hello</h1>")
}

func noop(w http.ResponseWriter, r *http.Request) {
	b := make([]byte, r.ContentLength)
	r.Body.Read(b)
	go wasteCPU()
	w.Write(b)
}

func wasteCPU() {
	sum := 0
	start := time.Now()
	for time.Since(start) < 10*time.Second {
		sum += rand.Intn(100)
	}
}

func main() {
	http.HandleFunc("/", index)
	http.HandleFunc("/noop", noop)

	log.Println("testapp server running on :3000")
	http.ListenAndServe(":3000", nil)
}