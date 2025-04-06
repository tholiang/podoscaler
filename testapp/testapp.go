package main

import (
	"context"
	"fmt"
	"math/rand/v2"
	"net/http"
	"time"
	"encoding/json"
	"io"
	"sync"
)

var wg sync.WaitGroup

func index(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "<h1>hello</h1>")
}

func sendRequests(w http.ResponseWriter, r *http.Request) {
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

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	for i := 0; i < numThreads; i++ {
		wg.Add(1)
		go wasteCPU(ctx)
	}

	fmt.Fprintf(w, fmt.Sprintf("Created %d goroutines.", numThreads))
	wg.Wait()
}

func wasteCPU(ctx context.Context) {
	defer wg.Done()
	
	for {
		select {
		case <-ctx.Done():
			return
		default:
			_ = rand.IntN(100)
		}
	}
}

func main() {
	http.HandleFunc("/", index)
	http.HandleFunc("/sendrequests", sendRequests)

	// log.Println("testapp server running on :3000")
	http.ListenAndServe(":3000", nil)
}
