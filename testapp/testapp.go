package main

import (
	"fmt"
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
	http.HandleFunc("/noop", noop)
	http.ListenAndServe(":3000", nil)
}
