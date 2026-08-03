package kernel_test

import (
	"context"
	"errors"
	"testing"

	"aisaaslab/internal/kernel"
)

var (
	errBlockedByGuard = errors.New("guard: subject restricted")
	errActionFailed   = errors.New("action: payment processor down")
)

func TestStateMachine_ValidTransitions(t *testing.T) {
	sm := kernel.NewStateMachine[kernel.State, string, string]()

	sm.AddTransition(kernel.NewTransition[kernel.State, string, string]("trial", "activate", "active")).
		AddTransition(kernel.NewTransition[kernel.State, string, string]("active", "payment_failed", "past_due")).
		AddTransition(kernel.NewTransition[kernel.State, string, string]("past_due", "payment_succeeded", "active")).
		AddTransition(kernel.NewTransition[kernel.State, string, string]("active", "cancel", "cancelled"))

	ctx := context.Background()

	tests := []struct {
		name    string
		current kernel.State
		event   string
		subject string
		want    kernel.State
		wantErr bool
	}{
		{
			name:    "trial to active",
			current: "trial",
			event:   "activate",
			subject: "key-123",
			want:    "active",
			wantErr: false,
		},
		{
			name:    "active to past_due",
			current: "active",
			event:   "payment_failed",
			subject: "key-123",
			want:    "past_due",
			wantErr: false,
		},
		{
			name:    "invalid event from cancelled",
			current: "cancelled",
			event:   "activate",
			subject: "key-123",
			want:    "cancelled",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := sm.Fire(ctx, tt.current, tt.event, tt.subject)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Fire() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("Fire() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStateMachine_CanFire(t *testing.T) {
	sm := kernel.NewStateMachine[string, string, any]()
	sm.AddTransition(kernel.NewTransition[string, string, any]("draft", "publish", "published"))

	if !sm.CanFire("draft", "publish") {
		t.Errorf("expected CanFire('draft', 'publish') to be true")
	}
	if sm.CanFire("draft", "archive") {
		t.Errorf("expected CanFire('draft', 'archive') to be false")
	}
}

func TestStateMachine_GuardClosure(t *testing.T) {
	sm := kernel.NewStateMachine[string, string, string]()

	restrictiveGuard := func(ctx context.Context, from, event, subject string) error {
		if subject == "banned-user" {
			return errBlockedByGuard
		}
		return nil
	}

	sm.AddTransition(
		kernel.NewTransition[string, string, string]("pending", "approve", "approved").
			WithGuard(restrictiveGuard),
	)

	ctx := context.Background()

	// Allowed user
	next, err := sm.Fire(ctx, "pending", "approve", "good-user")
	if err != nil {
		t.Fatalf("expected allowed user transition to succeed, got: %v", err)
	}
	if next != "approved" {
		t.Errorf("expected next state 'approved', got: %s", next)
	}

	// Banned user -> Guard must block
	stateBefore, err := sm.Fire(ctx, "pending", "approve", "banned-user")
	if !errors.Is(err, errBlockedByGuard) {
		t.Fatalf("expected errBlockedByGuard, got: %v", err)
	}
	if stateBefore != "pending" {
		t.Errorf("expected state to remain 'pending' after failed guard, got: %s", stateBefore)
	}
}

func TestStateMachine_ActionClosure(t *testing.T) {
	sm := kernel.NewStateMachine[string, string, string]()

	actionExecuted := false
	actionClosure := func(ctx context.Context, from, to, event, subject string) error {
		if subject == "fail-action" {
			return errActionFailed
		}
		actionExecuted = true
		return nil
	}

	sm.AddTransition(
		kernel.NewTransition[string, string, string]("created", "process", "processed").
			WithAction(actionClosure),
	)

	ctx := context.Background()

	// Successful action
	next, err := sm.Fire(ctx, "created", "process", "valid-item")
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if next != "processed" {
		t.Errorf("expected state 'processed', got: %s", next)
	}
	if !actionExecuted {
		t.Errorf("expected action closure to execute")
	}

	// Action failure -> State state unchanged
	stateBefore, err := sm.Fire(ctx, "created", "process", "fail-action")
	if !errors.Is(err, errActionFailed) {
		t.Fatalf("expected errActionFailed, got: %v", err)
	}
	if stateBefore != "created" {
		t.Errorf("expected state to remain 'created' after failed action, got: %s", stateBefore)
	}
}
