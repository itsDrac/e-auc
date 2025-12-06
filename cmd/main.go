package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/itsDrac/e-auc/cmd/web"
)

func main() {
	// Service initialization
	fmt.Println("Initializing e-auction service...")
	// Application entry point
	handler := web.NewChiHandler()
	handler.Mount()

	// Initialize the application
	serv := http.Server{
		Addr:    ":8080",
		Handler: handler.GetMux(),
	}
	fmt.Printf("Starting server on %s\n", serv.Addr)
	go serv.ListenAndServe()
	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, os.Interrupt, syscall.SIGTERM)
	<-shutdownChan
	serv.Close()
	fmt.Println("\nClosed server gracefully")
}