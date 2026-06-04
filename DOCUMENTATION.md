# Code Breaker Game

A TCP-based multiplayer guessing game. Two players join a shared session and take turns guessing a server-generated 4-digit secret code.

## Running

```bash
go run main.go server                      # start the server
go run main.go client [host:port]          # connect a client (default: localhost:8080)
go test ./game/... -v                      # run tests
```

---

## Game Rules

1. Two players connect; the server pairs them into a shared session.
2. Players alternate guessing a secret 4-digit code (1000–9999).
3. After each guess both players receive feedback.
4. The first player to guess correctly wins. Both are then offered a rematch.

---

## Secret Code Generation

A random 4-digit number is transformed by these rules:

| Condition | Action |
|---|---|
| Digit sum is **even** | Reverse the number (`1234` → `4321`) |
| Digit sum is **odd** | Increment each digit by 1, wrapping `9` → `0` |
| Result is a **palindrome** | Replace with `7777` |
| Result has a **leading zero** | Discard and re-generate |

**Difficulty levels** (server uses `Medium`):

| Level | Extra constraint |
|---|---|
| `Easy` | No repeated digits |
| `Medium` | None |
| `Hard` | At least one repeated digit and digit sum is prime |

---

## Feedback

Each response includes:
- Digits **correct and in the right position**
- Digits **correct but misplaced**
- A positional hint (first vs second half) when applicable

```
TIME: 1746000000 Correct digits in right position: 1. Correct digits in wrong position: 2. Hint: ...
```

---

## Architecture

```
main.go
  ├── server → StartServer()
  │     └── lobby: pairs two connections into a GameSession
  │           ├── startSession() — generates secret, manages rematch loop
  │           └── runPlayerLoop() (goroutine per player)
  │                 ├── turn-based via buffered channels
  │                 ├── ValidateGuess / GenerateFeedback / GenerateTimestampPrefix
  │                 └── broadcasts state to both players after each guess
  └── client → StartClient()
        └── event-driven loop: prompts for input only on YOUR_TURN
```

- Newline-framed TCP (`bufio.Scanner` / `fmt.Fprintln`).
- Per-connection write mutexes prevent interleaved messages.
- Session state (`done`, turn channels) is goroutine-safe.
