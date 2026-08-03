package kernel

import (
	"context"
	"fmt"
)

type State string

// GuardFunc evaluates pre-conditions before a transition occurs.
// Returning a non-nil error aborts the transition with that specific error.
type GuardFunc[S comparable, E comparable, T any] func(ctx context.Context, from S, event E, subject T) error

// ActionFunc executes side effects during a state transition before the new state is committed.
// Returning a non-nil error aborts the transition and state change.
type ActionFunc[S comparable, E comparable, T any] func(ctx context.Context, from S, to S, event E, subject T) error

// Transition describes one legal move in the state machine.
// Encapsulates routing (From -> Event -> To), pre-condition closures (Guards),
// and transactional side-effect closures (Action).
type Transition[S comparable, E comparable, T any] struct {
	From   S
	Event  E
	To     S
	Guards []GuardFunc[S, E, T]
	Action ActionFunc[S, E, T]
}

// NewTransition creates a basic transition route.
func NewTransition[S comparable, E comparable, T any](from S, event E, to S) Transition[S, E, T] {
	return Transition[S, E, T]{
		From:  from,
		Event: event,
		To:    to,
	}
}

// WithGuard appends one or more guard closures to the transition.
func (t Transition[S, E, T]) WithGuard(guards ...GuardFunc[S, E, T]) Transition[S, E, T] {
	t.Guards = append(t.Guards, guards...)
	return t
}

// WithAction attaches a side-effect execution closure to the transition.
func (t Transition[S, E, T]) WithAction(action ActionFunc[S, E, T]) Transition[S, E, T] {
	t.Action = action
	return t
}

type stateKey[S comparable, E comparable] struct {
	From  S
	Event E
}

// StateMachine is a thread-safe, generic, O(1) hashmap-backed state machine.
type StateMachine[S comparable, E comparable, T any] struct {
	transitions map[stateKey[S, E]]Transition[S, E, T]
}

func NewStateMachine[S comparable, E comparable, T any]() *StateMachine[S, E, T] {
	return &StateMachine[S, E, T]{
		transitions: make(map[stateKey[S, E]]Transition[S, E, T]),
	}
}

// AddTransition registers a transition closure rule in O(1) map indexing.
func (sm *StateMachine[S, E, T]) AddTransition(t Transition[S, E, T]) *StateMachine[S, E, T] {
	key := stateKey[S, E]{From: t.From, Event: t.Event}
	sm.transitions[key] = t
	return sm
}

// CanFire returns true if a transition exists for the given state and event.
func (sm *StateMachine[S, E, T]) CanFire(current S, event E) bool {
	key := stateKey[S, E]{From: current, Event: event}
	_, exists := sm.transitions[key]
	return exists
}

// Fire attempts to execute a state transition:
// 1. Validates structural transition existence in O(1) time.
// 2. Evaluates all Guard closures; fails fast if any guard returns an error.
// 3. Executes the Action closure (if defined) for transactional side-effects.
// 4. Returns the target state upon success.
func (sm *StateMachine[S, E, T]) Fire(ctx context.Context, current S, event E, subject T) (S, error) {
	key := stateKey[S, E]{From: current, Event: event}
	t, exists := sm.transitions[key]
	if !exists {
		return current, fmt.Errorf("no valid transition from state %v on event %v", current, event)
	}

	for _, guard := range t.Guards {
		if guard != nil {
			if err := guard(ctx, current, event, subject); err != nil {
				return current, fmt.Errorf("transition guard failed [%v -> %v]: %w", current, t.To, err)
			}
		}
	}

	if t.Action != nil {
		if err := t.Action(ctx, current, t.To, event, subject); err != nil {
			return current, fmt.Errorf("transition action failed [%v -> %v]: %w", current, t.To, err)
		}
	}

	return t.To, nil
}
