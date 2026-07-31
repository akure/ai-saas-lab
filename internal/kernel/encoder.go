package kernel

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"fmt"
)

// Encoder is the strategy interface every wire format implements. Modules
// never call json.Marshal or gob.NewEncoder directly — they ask the App for
// "the encoder named X" and use it. That indirection is what lets you swap
// or add formats without touching the modules that use them.
type Encoder interface {
	Encode(v any) ([]byte, error)
	Decode(data []byte, v any) error
}

// --- JSON: what CompletionModule uses for its public HTTP API ---

type jsonEncoder struct{}

func (jsonEncoder) Encode(v any) ([]byte, error) { return json.Marshal(v) }
func (jsonEncoder) Decode(d []byte, v any) error { return json.Unmarshal(d, v) }

// --- Gob: stdlib binary codec standing in for a "protobuf-style" internal
// wire format. Real production code would swap this for google.golang.org/protobuf;
// this lab keeps it dependency-free so it builds with zero network access,
// but the registration pattern below is identical either way. ---

type gobEncoder struct{}

func (gobEncoder) Encode(v any) ([]byte, error) {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (gobEncoder) Decode(d []byte, v any) error {
	return gob.NewDecoder(bytes.NewReader(d)).Decode(v)
}

func (a *App) RegisterEncoder(name string, enc Encoder) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.encoders[name] = enc
}

func (a *App) Encoder(name string) (Encoder, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	enc, ok := a.encoders[name]
	if !ok {
		return nil, fmt.Errorf("no encoder registered: %q", name)
	}
	return enc, nil
}

// RegisterDefaultEncoders wires up the two codecs this lab ships with.
func RegisterDefaultEncoders(a *App) {
	a.RegisterEncoder("json", jsonEncoder{})
	a.RegisterEncoder("gob", gobEncoder{})
}
