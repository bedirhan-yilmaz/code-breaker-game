package game

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
)

func StartClient(address string) error {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return fmt.Errorf("error connecting to server: %v", err)
	}
	defer conn.Close()

	fmt.Println("Welcome to the Code Breaker Game! Enter a code between 1000 and 9999.")

	reader := bufio.NewReader(os.Stdin)
	scanner := bufio.NewScanner(conn)

	for scanner.Scan() {
		msg := scanner.Text()

		switch {
		case msg == "YOUR_TURN":
			fmt.Print("\nEnter your guess (secret code) or 'exit' to quit: ")
			input, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("error reading input: %v", err)
			}
			input = strings.TrimRight(input, "\r\n")

			if input == "exit" {
				fmt.Println("Exiting the game.")
				return nil
			}

			if _, err := fmt.Fprintln(conn, input); err != nil {
				return fmt.Errorf("error sending message to server: %v", err)
			}

		case strings.HasPrefix(msg, "REMATCH?"):
			fmt.Println("\n" + msg)
			fmt.Print("Your choice: ")
			input, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("error reading input: %v", err)
			}
			input = strings.TrimRight(input, "\r\n")

			if _, err := fmt.Fprintln(conn, input); err != nil {
				return fmt.Errorf("error sending message to server: %v", err)
			}

		case strings.HasPrefix(msg, "TIME:"):
			printResponse(msg)

		case strings.HasPrefix(msg, "OBSERVER "):
			// Strip the "OBSERVER " prefix and reuse printResponse for consistent formatting.
			fmt.Println("\n--- Opponent's Turn ---")
			printResponse(strings.TrimPrefix(msg, "OBSERVER "))

		case msg == "REMATCH_DECLINED Game over.",
			msg == "OPPONENT_DISCONNECTED Game over.":
			fmt.Println("\n" + msg)
			return nil

		case strings.HasPrefix(msg, "GAME_OVER"):
			fmt.Println("\n" + msg)

		default:
			// PLAYER 1/2, WAITING_FOR_OPPONENT, GAME_START, WAIT ..., OBSERVER ..., errors
			fmt.Println("\n" + msg)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading from server: %v", err)
	}
	return nil
}

func printResponse(response string) {
	fmt.Println("\n--- Server Response ---")

	if strings.HasPrefix(response, "TIME: ") {
		parts := strings.SplitN(response, " ", 3)
		if len(parts) >= 2 {
			fmt.Println("Time:    ", parts[1])
			response = strings.TrimSpace(strings.Join(parts[2:], " "))
		}
	}

	for _, part := range strings.Split(response, " Hint:") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if strings.HasPrefix(part, "Correct") || strings.HasSuffix(part, "!") {
			fmt.Println("Result:  ", part)
		} else {
			fmt.Println("Hint:    ", part)
		}
	}

	fmt.Println("-----------------------")
}
