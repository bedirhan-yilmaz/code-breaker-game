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

---

## Deployment

### Prerequisites

| Software | Version | Install |
|---|---|---|
| Docker | 20.10+ | https://docs.docker.com/get-docker/ |
| Docker Compose | v2 (bundled with Docker Desktop) | included with Docker Desktop |
| Git | any | https://git-scm.com |
| kubectl | any | https://kubernetes.io/docs/tasks/tools/ *(for Kubernetes only)* |
| kind | any | https://kind.sigs.k8s.io/docs/user/quick-start/#installation *(for Kubernetes only)* |

Verify your install:
```bash
docker --version
docker compose version
kubectl version --client
git --version
```

---

### 1. Clone the Repository

```bash
git clone <repository-url>
cd code-breaker-game
```

---

### 2. Build the Docker Image

```bash
docker build -t code-breaker-game .
```

This produces a ~10 MB image using a two-stage build:
- **Stage 1 (`golang:1.20-alpine`)** — compiles a static binary
- **Stage 2 (`alpine:3.19`)** — runs the binary with minimal overhead

Verify the image was created:
```bash
docker images code-breaker-game
```

---

### 3. Run the Server Container

```bash
docker run --rm -p 8080:8080 --name code-breaker-game-server code-breaker-game
```

| Flag | Purpose |
|---|---|
| `--rm` | Remove container on exit |
| `-p 8080:8080` | Map host port 8080 → container port 8080 |
| `--name code-breaker-game-server` | Named container for easier reference |

Expected output:
```
Server started, waiting for players...
```

---

### 4. Connect Clients and Play

Open **two separate terminals** and run:

```bash
# Terminal 1 — Player 1
go run main.go client localhost:8080

# Terminal 2 — Player 2
go run main.go client localhost:8080
```

**Expected flow:**
- Player 1 sees: `PLAYER 1` → `WAITING_FOR_OPPONENT`
- Player 2 connects → both see `GAME_START`
- Player 1 sees `YOUR_TURN`, Player 2 sees `WAIT P1 is guessing...`
- Players alternate guessing until the code is broken
- After each game: both are prompted `REMATCH? Type 'rematch' to play again or 'exit' to quit.`

---

### 5. Docker Compose Setup

```bash
# Terminal 1 — start the server in the background
docker compose up server -d

# Terminal 2 — Player 1
docker compose run --rm client

# Terminal 3 — Player 2
docker compose run --rm client
```

To stop the server:
```bash
docker compose down
```

---

### 6. Kubernetes Deployment

#### Prerequisites

- [kind](https://kind.sigs.k8s.io/docs/user/quick-start/#installation)
- [kubectl](https://kubernetes.io/docs/tasks/tools/)

#### 1. Create the cluster

```bash
kind create cluster --name code-breaker-game-dev
```

#### 2. Build & load the image

```bash
docker build -t code-breaker-game:latest .
kind load docker-image code-breaker-game:latest --name code-breaker-game-dev
```

#### 3. Deploy

```bash
kubectl apply -f k8s/namespace.yaml
kubectl apply -f k8s/
```

#### 4. Verify

```bash
kubectl get pods -n code-breaker-game
kubectl logs -n code-breaker-game deploy/code-breaker-game-server
```

#### 5. Connect and play

```bash
kubectl port-forward -n code-breaker-game svc/code-breaker-game-server-svc 8080:8080
```

Then in two separate terminals:

```bash
go run main.go client localhost:8080
```

#### 6. Teardown

```bash
kubectl delete namespace code-breaker-game
kind delete cluster --name code-breaker-game-dev
```

---

## Quick Reference

```bash
# Build
docker build -t code-breaker-game:latest .

# Run server (Docker)
docker run --rm -p 8080:8080 --name code-breaker-game-server code-breaker-game

# Run client (from host)
go run main.go client localhost:8080

# Compose: start server
docker compose up server -d

# Compose: launch a client (run once per player terminal)
docker compose run --rm client

# Compose: stop server
docker compose down

# Kubernetes: create cluster
kind create cluster --name code-breaker-game-dev

# Kubernetes: build & load image
kind load docker-image code-breaker-game:latest --name code-breaker-game-dev

# Kubernetes: deploy
kubectl apply -f k8s/namespace.yaml && kubectl apply -f k8s/

# Kubernetes: port-forward
kubectl port-forward -n code-breaker-game svc/code-breaker-game-server-svc 8080:8080

# Kubernetes: teardown
kubectl delete namespace code-breaker-game
kind delete cluster --name code-breaker-game-dev
```
