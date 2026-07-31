package kernel

import (
	"context"
	"fmt"
)

type State string

// Transition describes one legal move. If a (From, Event) pair isn't in the
// table, that move is not just "unchecked" — it's structurally impossible.
type Transition struct {
	From  State
	Event string
	To    State
	Guard func(ctx context.Context, subject any) bool // optional extra condition
}

type StateMachine struct {
	transitions []Transition
}

func NewStateMachine() *StateMachine { return &StateMachine{} }

// AddTransition returns the machine itself so calls can be chained.
func (sm *StateMachine) AddTransition(t Transition) *StateMachine {
	sm.transitions = append(sm.transitions, t)
	return sm
}

// Fire looks for a matching (current state, event) transition whose guard
// (if any) passes, and returns the resulting state.
func (sm *StateMachine) Fire(ctx context.Context, current State, event string, subject any) (State, error) {
	for _, t := range sm.transitions {
		if t.From != current || t.Event != event {
			continue
		}
		if t.Guard != nil && !t.Guard(ctx, subject) {
			continue
		}
		return t.To, nil
	}
	return current, fmt.Errorf("no valid transition from %q on event %q", current, event)
}
