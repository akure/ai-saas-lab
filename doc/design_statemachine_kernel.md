# State Machine Engine: Architecture & Operational Philosophy

## 1. Overview
The `StateMachine` module located in [`internal/kernel/statemachine.go`](file:///d:/AK-FILES/Golang%20April/ai-saas-lab/ai-saas-lab/internal/kernel/statemachine.go) is a generic, high-performance, $O(1)$ hashmap-backed state machine engine written in Go. It enforces valid state transitions, pre-condition checks (guards), and transactional side-effects (actions) for domain objects across the application.

---

## 2. Core Architectural Principles & Operational Philosophy

### 2.1 Stateless & Immutable Execution Model
Unlike traditional state machines that store an entity's current state inside the engine instance, this `StateMachine` is designed to be completely **stateless**:
- **No Internal Entity State:** The [`StateMachine`](file:///d:/AK-FILES/Golang%20April/ai-saas-lab/ai-saas-lab/internal/kernel/statemachine.go#L59) struct only stores transition routing rules in its `transitions` map.
- **External State Ownership:** Entity state remains owned by domain models or database records.
- **Pure Function Semantics:** The [`Fire`](file:///d:/AK-FILES/Golang%20April/ai-saas-lab/ai-saas-lab/internal/kernel/statemachine.go#L88) function receives the entity's `current` state as an input argument and returns the target `to` state without modifying internal state machine memory.

### 2.2 Zero-Lock Concurrency & Thread-Safety Design
Thread safety is achieved by dividing operation into two explicit lifecycle phases:

```
+-------------------------------------------------------------+
|                      1. BOOTSTRAP PHASE                     |
|           Single-threaded setup at app initialization       |
|            sm.AddTransition(NewTransition(...))             |
+-------------------------------------------------------------+
                               |
                               v (Transitions map becomes read-only)
+-------------------------------------------------------------+
|                       2. RUNTIME PHASE                      |
|          Multi-threaded concurrent execution across workers |
|               sm.CanFire(...)  /  sm.Fire(...)              |
|        (Zero locks required - 100% safe Go map reads)       |
+-------------------------------------------------------------+
```

1. **Bootstrap Phase (Write-Only Setup):**
   - At application boot, transition rules are populated sequentially using [`AddTransition`](file:///d:/AK-FILES/Golang%20April/ai-saas-lab/ai-saas-lab/internal/kernel/statemachine.go#L70).
2. **Runtime Phase (Read-Only Concurrent Execution):**
   - Once initialized, the internal `transitions` map becomes **immutable and read-only**.
   - Concurrent requests calling [`CanFire`](file:///d:/AK-FILES/Golang%20April/ai-saas-lab/ai-saas-lab/internal/kernel/statemachine.go#L77) and [`Fire`](file:///d:/AK-FILES/Golang%20April/ai-saas-lab/ai-saas-lab/internal/kernel/statemachine.go#L88) only perform read lookups (`_, exists := sm.transitions[key]`).
   - According to the Go Memory Model, concurrent map reads are **100% thread-safe** without needing synchronization locks (`sync.Mutex` or `sync.RWMutex`), providing maximum runtime execution throughput.

### 2.3 Transactional Guard & Action Lifecycle
Each transition execution follows a strict 4-step pipeline in [`Fire`](file:///d:/AK-FILES/Golang%20April/ai-saas-lab/ai-saas-lab/internal/kernel/statemachine.go#L88):

1. **Structural Validation:** Evaluates structural route existence in $O(1)$ time. Fails fast if no rule exists for `(current, event)`.
2. **Guard Evaluation:** Sequentially executes registered [`GuardFunc`](file:///d:/AK-FILES/Golang%20April/ai-saas-lab/ai-saas-lab/internal/kernel/statemachine.go#L12) closures. If any guard fails, the transition is aborted immediately and the state remains unchanged.
3. **Action Execution:** Executes the optional [`ActionFunc`](file:///d:/AK-FILES/Golang%20April/ai-saas-lab/ai-saas-lab/internal/kernel/statemachine.go#L16) closure for side-effects before committing the state. If the action returns an error, the transition is aborted.
4. **State Commit:** Returns the target `To` state upon successful completion of guards and action.

### 2.4 Fluent Builder with Value Semantics
The [`Transition`](file:///d:/AK-FILES/Golang%20April/ai-saas-lab/ai-saas-lab/internal/kernel/statemachine.go#L21) struct uses value semantics:
- Builder methods [`WithGuard`](file:///d:/AK-FILES/Golang%20April/ai-saas-lab/ai-saas-lab/internal/kernel/statemachine.go#L41) and [`WithAction`](file:///d:/AK-FILES/Golang%20April/ai-saas-lab/ai-saas-lab/internal/kernel/statemachine.go#L48) modify and return struct copies.
- Enables fluent chaining while eliminating unintended state leakage across route definitions.

---

## 3. Data Structures & Type System

```go
type stateKey[S comparable, E comparable] struct {
    From  S
    Event E
}

type StateMachine[S comparable, E comparable, T any] struct {
    transitions map[stateKey[S, E]]Transition[S, E, T]
}
```

- **`S` (State Type):** Must satisfy `comparable` constraint (e.g., `string`, `int`, custom enum).
- **`E` (Event Type):** Must satisfy `comparable` constraint (e.g., `string`, custom event enum).
- **`T` (Subject Type):** Represents the payload or domain entity type (`any`) passed to guards and actions.

---

## 4. Operational Guidelines & Anti-Patterns

### ✅ Do
- Register all transition rules during single-threaded application startup.
- Ensure guard closures are idempotent and free of unexpected side effects.
- Pass `context.Context` down to guards and actions to support cancellation and timeouts.

### ❌ Don't
- **Never modify transitions at runtime:** Invoking `AddTransition` while concurrent goroutines call `Fire` will cause a Go runtime map panic (`fatal error: concurrent map writes and map reads`).
- **Do not store state in the engine:** Keep domain state in persistent storage or domain struct fields.

---

## 5. Code Example

```go
package main

import (
    "context"
    "fmt"
    "aisaaslab/internal/kernel"
)

type State string
type Event string

type Account struct {
    ID     string
    Active bool
}

func main() {
    // 1. Instantiate state machine
    sm := kernel.NewStateMachine[State, Event, *Account]()

    // Guard closure
    checkActive := func(ctx context.Context, from State, event Event, acc *Account) error {
        if !acc.Active {
            return fmt.Errorf("account %s is inactive", acc.ID)
        }
        return nil
    }

    // 2. Bootstrap Phase: Register transitions
    sm.AddTransition(
        kernel.NewTransition[State, Event, *Account]("trial", "upgrade", "pro").
            WithGuard(checkActive),
    )

    // 3. Runtime Phase: Concurrent execution
    acc := &Account{ID: "acc_456", Active: true}
    nextState, err := sm.Fire(context.Background(), "trial", "upgrade", acc)
    if err != nil {
        fmt.Printf("Transition failed: %v\n", err)
        return
    }

    fmt.Printf("Transition successful! New State: %s\n", nextState) // Outputs: pro
}
```
