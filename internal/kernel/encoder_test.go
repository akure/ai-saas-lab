package kernel

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

type samplePayload struct {
	Name  string   `json:"name"`
	Age   int      `json:"age"`
	Items []string `json:"items"`
}

func TestRegisterDefaultEncoders_RegistersJSONAndGob(t *testing.T) {
	app := NewApp(&Config{})

	RegisterDefaultEncoders(app)

	jsonEnc, err := app.Encoder("json")
	if err != nil {
		t.Fatalf("expected json encoder to be registered: %v", err)
	}
	if _, ok := jsonEnc.(jsonEncoder); !ok {
		t.Fatalf("expected json encoder to be of type jsonEncoder, got %T", jsonEnc)
	}

	gobEnc, err := app.Encoder("gob")
	if err != nil {
		t.Fatalf("expected gob encoder to be registered: %v", err)
	}
	if _, ok := gobEnc.(gobEncoder); !ok {
		t.Fatalf("expected gob encoder to be of type gobEncoder, got %T", gobEnc)
	}
}

func TestEncoder_UnknownNameReturnsError(t *testing.T) {
	app := NewApp(&Config{})

	_, err := app.Encoder("missing")
	if err == nil {
		t.Fatal("expected error for missing encoder, got nil")
	}
	if !strings.Contains(err.Error(), "no encoder registered") {
		t.Fatalf("expected registration error, got %v", err)
	}
}

func TestRegisterEncoder_OverridesExistingRegistration(t *testing.T) {
	app := NewApp(&Config{})

	first := fakeEncoder{}
	app.RegisterEncoder("json", first)

	second := fakeEncoder{}
	app.RegisterEncoder("json", second)

	enc, err := app.Encoder("json")
	if err != nil {
		t.Fatalf("expected encoder lookup to succeed: %v", err)
	}
	if got := reflect.TypeOf(enc); got != reflect.TypeOf(second) {
		t.Fatalf("expected overridden encoder to be used, got %T", enc)
	}
}

func TestJSONEncoder_EncodeDecodeRoundTrip(t *testing.T) {
	enc := jsonEncoder{}
	payload := samplePayload{Name: "alice", Age: 30, Items: []string{"one", "two"}}

	data, err := enc.Encode(payload)
	if err != nil {
		t.Fatalf("json encode failed: %v", err)
	}

	var decoded samplePayload
	if err := enc.Decode(data, &decoded); err != nil {
		t.Fatalf("json decode failed: %v", err)
	}

	if decoded.Name != payload.Name || decoded.Age != payload.Age || !reflect.DeepEqual(decoded.Items, payload.Items) {
		t.Fatalf("round trip mismatch: got %+v want %+v", decoded, payload)
	}
}

func TestJSONEncoder_DecodeInvalidJSONReturnsError(t *testing.T) {
	enc := jsonEncoder{}
	var decoded samplePayload

	err := enc.Decode([]byte("not-valid-json"), &decoded)
	if err == nil {
		t.Fatal("expected invalid json to return an error")
	}
}

func TestGobEncoder_EncodeDecodeRoundTrip(t *testing.T) {
	enc := gobEncoder{}
	payload := samplePayload{Name: "bob", Age: 25, Items: []string{"x", "y"}}

	data, err := enc.Encode(payload)
	if err != nil {
		t.Fatalf("gob encode failed: %v", err)
	}

	var decoded samplePayload
	if err := enc.Decode(data, &decoded); err != nil {
		t.Fatalf("gob decode failed: %v", err)
	}

	if decoded.Name != payload.Name || decoded.Age != payload.Age || !reflect.DeepEqual(decoded.Items, payload.Items) {
		t.Fatalf("round trip mismatch: got %+v want %+v", decoded, payload)
	}
}

func TestGobEncoder_DecodeInvalidBytesReturnsError(t *testing.T) {
	enc := gobEncoder{}
	var decoded samplePayload

	err := enc.Decode([]byte("not-valid-gob-data"), &decoded)
	if err == nil {
		t.Fatal("expected invalid gob data to return an error")
	}
}

func TestGobEncoder_EncodeNilValue(t *testing.T) {
	enc := gobEncoder{}
	_, err := enc.Encode(nil)
	if err == nil {
		t.Fatal("expected gob encode of nil to return an error")
	}
	if !strings.Contains(err.Error(), "cannot encode nil value") {
		t.Fatalf("expected gob nil-value error, got %v", err)
	}
}

func TestJSONEncoder_EncodeNilValue(t *testing.T) {
	enc := jsonEncoder{}
	data, err := enc.Encode(nil)
	if err != nil {
		t.Fatalf("expected json encode of nil to succeed, got %v", err)
	}
	if string(data) != "null" {
		t.Fatalf("expected json null payload, got %q", string(data))
	}
}

type fakeEncoder struct{}

func (fakeEncoder) Encode(v any) ([]byte, error)    { return json.Marshal(v) }
func (fakeEncoder) Decode(data []byte, v any) error { return json.Unmarshal(data, v) }
