package main

import (
	"fmt"
	"net/http"
	"os"
)

// this is a dumb service that just runs a hello-world style http server
func main() {
	http.HandleFunc("GET /{$}", http.HandlerFunc(getRoot))
	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8080" // default address
	}
	fmt.Println("Starting server on", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		if err != http.ErrServerClosed {
			panic(err) // panic if the server fails to start
		}
	}
}

func getRoot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("Hello, World!\nThis is svc1!\n"))
}
