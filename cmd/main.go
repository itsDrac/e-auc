package main

import (
	"fmt"
	"log"

	"github.com/itsDrac/e-auc/internal/server"
)

func main() {
	// Service initialization
	fmt.Println("Initializing e-auction service...")
	server := server.New()
	if err := server.Run(); err != nil {
		log.Fatal("server failed")
	}
}
