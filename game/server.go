package game

import (
	"fmt"
	"log"
	"net"
	"sync"
)

func StartServer() {
	listener, err := net.Listen("tcp", "0.0.0.0:8080")
	if err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
	defer listener.Close()

	fmt.Println("Server started, waiting for players...")

	var (
		lobbyMu       sync.Mutex
		waitingConn   net.Conn
		waitingSignal chan struct{}
	)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Error accepting connection: %v", err)
			continue
		}

		lobbyMu.Lock()
		if waitingConn == nil {
			// First player: park until a partner arrives
			waitingConn = conn
			waitingSignal = make(chan struct{})
			lobbyMu.Unlock()

			fmt.Println("Player 1 connected. Waiting for Player 2...")
			go func(c net.Conn, ready chan struct{}) {
				writeToClient(c, "PLAYER 1")
				writeToClient(c, "WAITING_FOR_OPPONENT")
				<-ready // blocks until partner arrives
			}(conn, waitingSignal)

		} else {
			// Second player: form the session
			p1 := waitingConn
			sig := waitingSignal
			waitingConn = nil
			waitingSignal = nil
			lobbyMu.Unlock()

			fmt.Println("Player 2 connected. Starting game session.")
			writeToClient(conn, "PLAYER 2")
			go startSession(p1, conn, sig)
		}
	}
}

func writeToClient(conn net.Conn, s string) {
	_, err := fmt.Fprintln(conn, s)
	if err != nil {
		log.Printf("Error writing to client: %v", err)
	}
}
