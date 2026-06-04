# CCS Code Breaker — Deployment Guide

## Prerequisites

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

## 1. Clone the Repository

```bash
git clone <repository-url>
cd ccs-interview/GO
```

---

## 2. Build the Docker Image

```bash
docker build -t ccs-game .
```

This produces a ~10 MB image using a two-stage build:
- **Stage 1 (`golang:1.20-alpine`)** — compiles a static binary
- **Stage 2 (`alpine:3.19`)** — runs the binary with minimal overhead

Verify the image was created:
```bash
docker images ccs-game
```

---

## 3. Run the Server Container

```bash
docker run --rm -p 8080:8080 --name ccs-server ccs-game
```

| Flag | Purpose |
|---|---|
| `--rm` | Remove container on exit |
| `-p 8080:8080` | Map host port 8080 → container port 8080 |
| `--name ccs-server` | Named container for easier reference |

Expected output:
```
Server started, waiting for players...
```

---

## 4. Connect Clients and Play

The client is a CLI application run directly on your machine (no Docker required for clients).

Open **two separate terminals** and run:

```bash
# Terminal 1 — Player 1
go run main.go client localhost:8080

# Terminal 2 — Player 2
go run main.go client localhost:8080
```

Or using the compiled binary if you have it:
```bash
./game client localhost:8080
```

**Expected flow:**
- Player 1 sees: `PLAYER 1` → `WAITING_FOR_OPPONENT`
- Player 2 connects → both see `GAME_START`
- Player 1 sees `YOUR_TURN`, Player 2 sees `WAIT P1 is guessing...`
- Players alternate guessing until the code is broken
- After each game: both are prompted `REMATCH? Type 'rematch' to play again or 'exit' to quit.`

---

## 5. Docker Compose Setup (Server + Clients Together)

```bash
# Terminal 1 — start the server in the background
docker compose up server -d

# Terminal 2 — Player 1 (launches a fresh interactive client container)
docker compose run --rm client

# Terminal 3 — Player 2 (launches a second independent client container)
docker compose run --rm client
```

`docker compose run` spawns a new container each time, giving each player their own interactive terminal. `--rm` removes the container automatically when the player exits.

To stop the server:
```bash
docker compose down
```

---

## 6. Kubernetes Deployment

### Prerequisites

- [kind](https://kind.sigs.k8s.io/docs/user/quick-start/#installation)
- [kubectl](https://kubernetes.io/docs/tasks/tools/)

### 1. Create the cluster

```bash
kind create cluster --name ccs-game-dev
```

### 2. Build & load the image

kind clusters cannot pull local Docker images directly — build and load the image first:

```bash
docker build -t ccs-game:latest .
kind load docker-image ccs-game:latest --name ccs-game-dev
```

### 3. Deploy

```bash
kubectl apply -f k8s/namespace.yaml
kubectl apply -f k8s/
```

### 4. Verify

```bash
kubectl get pods -n ccs-game
kubectl logs -n ccs-game deploy/ccs-server
```

### 5. Connect and play

Use `port-forward` to expose the server to your local machine:

```bash
kubectl port-forward -n ccs-game svc/ccs-server-svc 8080:8080
```

Then in two separate terminals:

```bash
# Terminal 1 — Player 1
go run main.go client localhost:8080

# Terminal 2 — Player 2
go run main.go client localhost:8080
```

### 6. Teardown

```bash
kubectl delete namespace ccs-game
kind delete cluster --name ccs-game-dev
```

---

## Quick Reference

```bash
# Build
docker build -t ccs-game:latest .

# Run server (Docker)
docker run --rm -p 8080:8080 --name ccs-server ccs-game

# Run client (from host)
go run main.go client localhost:8080

# Compose: start server
docker compose up server -d

# Compose: launch a client (run once per player terminal)
docker compose run --rm client

# Compose: stop server
docker compose down

# Kubernetes: create cluster
kind create cluster --name ccs-game-dev

# Kubernetes: build & load image
kind load docker-image ccs-game:latest --name ccs-game-dev

# Kubernetes: deploy
kubectl apply -f k8s/namespace.yaml && kubectl apply -f k8s/

# Kubernetes: port-forward
kubectl port-forward -n ccs-game svc/ccs-server-svc 8080:8080

# Kubernetes: teardown
kubectl delete namespace ccs-game
kind delete cluster --name ccs-game-dev
```
