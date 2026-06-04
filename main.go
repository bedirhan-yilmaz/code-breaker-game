package main

import (
	"ccs_interview/game"
	"log"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: go run main.go <mode>")
	}

	mode := os.Args[1]

	switch mode {
	case "server":
		game.StartServer()
	case "client":
		host := "localhost:8080"
		if len(os.Args) >= 3 {
			host = os.Args[2]
		}
		if err := game.StartClient(host); err != nil {
			log.Fatal(err)
		}
	default:
		log.Fatal("Invalid mode. Use 'server' or 'client'.")
	}
}
