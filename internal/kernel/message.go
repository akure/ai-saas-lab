package kernel

import (
	"context"
	"fmt"
)

// MessageDescriptor answers: "given a type name, what struct shape is this,
// and which registered encoder unpacks it?"
type MessageDescriptor struct {
	Type    string
	Encoder string     // must match a key registered via RegisterEncoder
	New     func() any // factory: produce a fresh zero-value target to decode into
}

// MessageHandler is what actually runs once a message has been decoded.
type MessageHandler func(ctx context.Context, msg any) error

func (a *App) RegisterMessage(desc MessageDescriptor) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.messages[desc.Type] = desc
}

func (a *App) RegisterHandler(msgType string, h MessageHandler) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.handlers[msgType] = h
}

// DecodeMessage turns raw bytes into a typed struct purely by looking up the
// message type name — the caller never needs to know if it's JSON or Gob.
func (a *App) DecodeMessage(msgType string, raw []byte) (any, error) {
	a.mu.RLock()
	desc, ok := a.messages[msgType]
	a.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("unknown message type: %q", msgType)
	}

	enc, err := a.Encoder(desc.Encoder)
	if err != nil {
		return nil, err
	}

	target := desc.New()
	if err := enc.Decode(raw, target); err != nil {
		return nil, fmt.Errorf("decode %q: %w", msgType, err)
	}
	return target, nil
}

// Dispatch decodes raw bytes by type name, then runs the registered handler
// for that type. This is the one function every transport (HTTP here, gRPC
// or Kafka in a bigger system) funnels through.
func (a *App) Dispatch(ctx context.Context, msgType string, raw []byte) error {
	msg, err := a.DecodeMessage(msgType, raw)
	if err != nil {
		return err
	}

	a.mu.RLock()
	handler, ok := a.handlers[msgType]
	a.mu.RUnlock()
	if !ok {
		return fmt.Errorf("no handler registered for message type: %q", msgType)
	}
	return handler(ctx, msg)
}
