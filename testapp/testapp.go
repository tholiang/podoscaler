package main

import (
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net/http"
)

func index(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "<h1>hello</h1>")
}

func noop(w http.ResponseWriter, r *http.Request) {
	b := make([]byte, r.ContentLength)
	r.Body.Read(b)
	w.Write(b)
}

func main() {
	http.HandleFunc("/", index)
	http.HandleFunc("/setthreads", setThreads)

	// log.Println("testapp server running on :3000")
	http.ListenAndServe(":3000", nil)
	// for {
	// 	go wasteCPU()
	// 	time.Sleep(100 * time.Millisecond)
	// }
}
