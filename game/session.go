package game

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
)

type playerID int

const (
	player1 playerID = 1
	player2 playerID = 2
)

// GameSession holds all shared state for a two-player game.
type GameSession struct {
	players    [2]net.Conn       // index 0 = player1, index 1 = player2
	scanners   [2]*bufio.Scanner // one per player, reused across restarts
	secretCode int

	writeMu [2]sync.Mutex // guards writes to players[i]; both goroutines write to the opponent's conn
	mu      sync.Mutex    // guards done
	done    bool

	turnCh [2]chan struct{} // buffered(1) turn tokens; holding the token means it's your turn
}

func (s *GameSession) opponentOf(id playerID) playerID {
	if id == player1 {
		return player2
	}
	return player1
}

// writeToPlayer serializes writes to a single connection.
func (s *GameSession) writeToPlayer(id playerID, msg string) {
	idx := int(id) - 1
	s.writeMu[idx].Lock()
	defer s.writeMu[idx].Unlock()
	writeToClient(s.players[idx], msg)
}

// gameResult carries the outcome of one complete game round.
type gameResult struct {
	winner playerID
	alive  bool // false if a player disconnected
}

// startSession is called once when both players have connected.
// It loops through games until players decline a rematch or disconnect.
func startSession(p1, p2 net.Conn, p1Ready chan struct{}) {
	session := &GameSession{
		players:  [2]net.Conn{p1, p2},
		scanners: [2]*bufio.Scanner{bufio.NewScanner(p1), bufio.NewScanner(p2)},
		turnCh:   [2]chan struct{}{make(chan struct{}, 1), make(chan struct{}, 1)},
	}

	close(p1Ready) // unblock player 1's lobby goroutine

	session.writeToPlayer(player1, "GAME_START")
	session.writeToPlayer(player2, "GAME_START")

	firstTurn := player1
	gameNum := 1
	for {
		session.secretCode = GenerateSecretCode(Medium)
		session.done = false
		fmt.Printf("[Game %d] Started. Secret code: %d. P%d goes first.\n", gameNum, session.secretCode, firstTurn)

		winner, alive := runGame(session, firstTurn)
		if !alive {
			fmt.Printf("[Game %d] Ended — player disconnected.\n", gameNum)
			return // a player disconnected mid-game
		}
		fmt.Printf("[Game %d] P%d wins!\n", gameNum, winner)

		if !askRematch(session) {
			fmt.Printf("[Game %d] No rematch. Session closed.\n", gameNum)
			session.players[0].Close()
			session.players[1].Close()
			return
		}

		fmt.Printf("[Game %d] Both players want a rematch.\n", gameNum)
		gameNum++
		// Loser goes first next round
		firstTurn = session.opponentOf(winner)
		session.writeToPlayer(player1, "GAME_START")
		session.writeToPlayer(player2, "GAME_START")
	}
}

// runGame executes one complete game (from first guess to win/disconnect).
func runGame(session *GameSession, firstTurn playerID) (playerID, bool) {
	resultCh := make(chan gameResult, 2)

	// Reset turn channels: drain any leftover tokens, then prime the first player.
	for i := range session.turnCh {
		select {
		case <-session.turnCh[i]:
		default:
		}
	}
	session.turnCh[int(firstTurn)-1] <- struct{}{}

	go runPlayerLoop(session, player1, resultCh)
	go runPlayerLoop(session, player2, resultCh)

	res := <-resultCh
	// Drain the second result in case both goroutines fired at once.
	select {
	case <-resultCh:
	default:
	}
	return res.winner, res.alive
}

// runPlayerLoop is the per-player goroutine for one game round.
func runPlayerLoop(session *GameSession, id playerID, resultCh chan<- gameResult) {
	idx := int(id) - 1
	opp := session.opponentOf(id)

	for {
		// Wait for our turn token (or channel close = game ended externally).
		if _, ok := <-session.turnCh[idx]; !ok {
			return
		}

		session.mu.Lock()
		if session.done {
			session.mu.Unlock()
			return
		}
		session.mu.Unlock()

		session.writeToPlayer(id, "YOUR_TURN")
		session.writeToPlayer(opp, fmt.Sprintf("WAIT P%d is guessing...", id))

		if !session.scanners[idx].Scan() {
			if err := session.scanners[idx].Err(); err != nil {
				log.Printf("P%d read error: %v", id, err)
			}
			handleDisconnect(session, id, resultCh)
			return
		}

		guess := session.scanners[idx].Text()
		fmt.Printf("[P%d] Guess: %s\n", id, guess)

		numGuess, err := ValidateGuess(guess)
		if err != nil {
			session.writeToPlayer(id, err.Error())
			// Invalid guess: stay on this player's turn.
			session.turnCh[idx] <- struct{}{}
			continue
		}

		prefix := GenerateTimestampPrefix()
		feedback := GenerateFeedback(session.secretCode, numGuess)

		session.writeToPlayer(id, prefix+" "+feedback)
		session.writeToPlayer(opp, fmt.Sprintf("OBSERVER %s P%d: %s", prefix, id, feedback))

		if strings.HasPrefix(feedback, "Congratulations") {
			session.mu.Lock()
			session.done = true
			session.mu.Unlock()

			fmt.Printf("[P%d] Guessed correctly: %d\n", id, numGuess)
			session.writeToPlayer(opp, fmt.Sprintf("GAME_OVER P%d wins! The code was %d.", id, session.secretCode))
			resultCh <- gameResult{winner: id, alive: true}
			return
		}

		fmt.Printf("[P%d] Feedback: %s\n", id, feedback)

		oppIdx := int(opp) - 1
		session.turnCh[oppIdx] <- struct{}{} // advance turn
	}
}

// handleDisconnect notifies the surviving player and ends the game round.
func handleDisconnect(session *GameSession, disconnectedID playerID, resultCh chan<- gameResult) {
	session.mu.Lock()
	if session.done {
		session.mu.Unlock()
		return
	}
	session.done = true
	session.mu.Unlock()

	survivor := session.opponentOf(disconnectedID)
	survivorIdx := int(survivor) - 1

	session.writeToPlayer(survivor, "OPPONENT_DISCONNECTED Game over.")
	fmt.Printf("[Session] P%d disconnected. Notifying P%d.\n", disconnectedID, survivor)
	session.players[0].Close()
	session.players[1].Close()
	// Unblock the survivor's goroutine if it's waiting on its turn channel.
	close(session.turnCh[survivorIdx])

	resultCh <- gameResult{winner: 0, alive: false}
}

// askRematch prompts both players and returns true only if both vote "rematch".
func askRematch(session *GameSession) bool {
	msg := "REMATCH? Type 'rematch' to play again or 'exit' to quit."
	session.writeToPlayer(player1, msg)
	session.writeToPlayer(player2, msg)

	votes := make(chan bool, 2)
	for _, id := range []playerID{player1, player2} {
		go func(pid playerID) {
			idx := int(pid) - 1
			if session.scanners[idx].Scan() &&
				strings.TrimSpace(strings.ToLower(session.scanners[idx].Text())) == "rematch" {
				votes <- true
			} else {
				votes <- false
			}
		}(id)
	}

	v1, v2 := <-votes, <-votes
	if v1 && v2 {
		return true
	}
	session.writeToPlayer(player1, "REMATCH_DECLINED Game over.")
	session.writeToPlayer(player2, "REMATCH_DECLINED Game over.")
	return false
}
