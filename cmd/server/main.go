package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	go exit()

	srv := NewServer()

	mux := http.NewServeMux()
	mux.HandleFunc("/", srv.rootHandler)

	server := &http.Server{
		Addr:              "localhost:8080",
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("metrics server listening on http://%s", server.Addr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("server error: %v", err)
	}
}

func exit() {
	// Create a channel to receive OS signals
	sig := make(chan os.Signal, 1)

	// Notify channel when user presses Ctrl+C (SIGINT) or termination signal (SIGTERM)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sig // Wait for signal
		fmt.Println("\nInterrupted. Exiting gracefully...")
		os.Exit(0) // Exit with code 0
	}()

	select {} // Block forever
}
