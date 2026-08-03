# AI SaaS Lab

AI SaaS Lab is a small, runnable MVP for building an AI-powered SaaS in Go. It brings together a modular application kernel with practical SaaS building blocks such as API-key auth, usage tracking, subscription handling, and billing-style workflows.

The goal is to provide a clear and extensible starting point for building in public without starting from a blank slate.

## What this project is

This repository is an MVP-style reference implementation for:

- an AI completion endpoint,
- API-key-based access control,
- usage recording and quota checks,
- subscription state handling,
- and a modular architecture that can grow over time.

It is intentionally lightweight and dependency-free so it can run locally and be understood quickly.

## Why this exists

This project is useful for three things:

1. Learning how a modular SaaS architecture can be built in Go.
2. Prototyping AI product ideas without setting up a full platform.
3. Building in public with a small but meaningful project that can evolve over time.

## Current capabilities

- Streaming chat-style completion flow
- API-key validation and policy checks
- Demo and admin-style API key creation
- Usage tracking
- Subscription state transitions
- Event-driven communication between modules
- A small kernel for app wiring, registration, and lifecycle management

## Architecture overview

The project is organized around a small kernel plus feature modules:

- kernel: app wiring, config, policies, event bus, encoders, store
- auth: API key validation and key issuance support
- completion: the AI-facing request flow
- billing: usage and subscription-related logic

## Run locally

### 1. Start the Server Backend
```bash
go run ./cmd/lab
```
The application starts on port 8080 by default.

### 2. Launch the Terminal UI Client (TUI)
```bash
go run ./cmd/tui
```
This launches a Linux-style terminal dashboard for real-time monitoring, multi-turn chat completion, persona switching, and session management across Windows, Linux, and macOS.


## Example usage

### 1. Create an API key

**Bash:**
```bash
curl -X POST http://localhost:8080/admin/api-keys \
  -H "Content-Type: application/json" \
  -d '{"plan":"pro"}'
```

**PowerShell:**
```powershell
Invoke-RestMethod -Uri "http://localhost:8080/admin/api-keys" -Method Post -ContentType "application/json" -Body '{"plan":"pro"}'
```

### 2. Call the completion endpoint

**Bash:**
```bash
curl -N -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{"api_key":"YOUR_API_KEY","prompt":"hello from the lab"}'
```

**PowerShell:**
```powershell
Invoke-RestMethod -Uri "http://localhost:8080/v1/chat/completions" -Method Post -ContentType "application/json" -Body '{"api_key":"YOUR_API_KEY","prompt":"hello from the lab"}'
```

### 3. Check usage

**Bash:**
```bash
curl http://localhost:8080/v1/usage/YOUR_API_KEY
```

**PowerShell:**
```powershell
Invoke-RestMethod -Uri "http://localhost:8080/v1/usage/YOUR_API_KEY"
```

## Project structure

- [cmd/lab/main.go](cmd/lab/main.go)
- [internal/kernel](internal/kernel)
- [internal/modules/auth](internal/modules/auth)
- [internal/modules/completion](internal/modules/completion)
- [internal/modules/billing](internal/modules/billing)

## MVP status

This is an MVP-style project, not a full enterprise platform. It is intentionally simple and educational while still being structured in a way that can grow into something more production-oriented later.

The current implementation uses:

- an in-memory store for local development,
- a simple event bus for decoupled module communication,
- and mocked completion behavior for the lab environment.

That keeps the project easy to understand and run while leaving room for future evolution.

## Contributing

Contributions are welcome. If you want to improve the architecture, add features, or make the platform more realistic, feel free to open an issue or submit a pull request.

## License

This project is intended for learning, experimentation, and open development. If you plan to use it publicly or commercially, please review the license terms before doing so.
