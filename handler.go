package main

import (
	"net/http"
	"strings"
)

func handler(w http.ResponseWriter, r *http.Request) {
	backend, ok := domains[r.Host]
	if !ok {
		w.Write([]byte("No route found"))
		return
	}

	targets, ok := backends[backend]
	if !ok {
		w.Write([]byte("Invalid route"))
		return
	}

	w.Write([]byte(strings.Join(targets, "\n")))
}
