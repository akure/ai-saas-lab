package kernel

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// ---------------------------------------------------------------------------
// 1. Encapsulated Smart Struct Types
// ---------------------------------------------------------------------------

// TenantKey represents a validated, non-empty tenant/customer identifier.
type TenantKey struct {
	value string
}

// NewTenantKey constructs a TenantKey, validating that it is non-empty.
func NewTenantKey(raw string) (TenantKey, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return TenantKey{}, errors.New("tenant_key cannot be empty")
	}
	return TenantKey{value: trimmed}, nil
}

// MustTenantKey constructs a TenantKey or panics if invalid (for constants/tests).
func MustTenantKey(raw string) TenantKey {
	k, err := NewTenantKey(raw)
	if err != nil {
		panic(err)
	}
	return k
}

func (t TenantKey) String() string { return t.value }
func (t TenantKey) IsZero() bool    { return t.value == "" }

func (t TenantKey) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.value)
}

func (t *TenantKey) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	if s == "" {
		*t = TenantKey{}
		return nil
	}
	k, err := NewTenantKey(s)
	if err != nil {
		return err
	}
	*t = k
	return nil
}

// APIKey represents a sensitive access token with automatic log-redaction safety.
type APIKey struct {
	value string
}

// NewAPIKey constructs an APIKey, validating that it is non-empty.
func NewAPIKey(raw string) (APIKey, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return APIKey{}, errors.New("api_key cannot be empty")
	}
	return APIKey{value: trimmed}, nil
}

// MustAPIKey constructs an APIKey or panics if invalid.
func MustAPIKey(raw string) APIKey {
	k, err := NewAPIKey(raw)
	if err != nil {
		panic(err)
	}
	return k
}

// String implements fmt.Stringer to ensure raw API keys are NEVER logged in plaintext.
func (k APIKey) String() string {
	return k.Redacted()
}

// Redacted returns a safe-for-logging representation (e.g. "sk-p...8f2a").
func (k APIKey) Redacted() string {
	if k.value == "" {
		return "<empty-key>"
	}
	if len(k.value) <= 8 {
		return "***"
	}
	return k.value[:4] + "..." + k.value[len(k.value)-4:]
}

// Raw returns the underlying plaintext secret key for authorized verification.
func (k APIKey) Raw() string {
	return k.value
}

func (k APIKey) IsZero() bool { return k.value == "" }

func (k APIKey) MarshalJSON() ([]byte, error) {
	return json.Marshal(k.value)
}

func (k *APIKey) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	if s == "" {
		*k = APIKey{}
		return nil
	}
	key, err := NewAPIKey(s)
	if err != nil {
		return err
	}
	*k = key
	return nil
}

// SubscriptionID represents a validated subscription instance identifier.
type SubscriptionID struct {
	value string
}

func NewSubscriptionID(raw string) (SubscriptionID, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return SubscriptionID{}, errors.New("subscription_id cannot be empty")
	}
	return SubscriptionID{value: trimmed}, nil
}

func MustSubscriptionID(raw string) SubscriptionID {
	s, err := NewSubscriptionID(raw)
	if err != nil {
		panic(err)
	}
	return s
}

func (s SubscriptionID) String() string { return s.value }
func (s SubscriptionID) IsZero() bool    { return s.value == "" }

func (s SubscriptionID) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.value)
}

func (s *SubscriptionID) UnmarshalJSON(b []byte) error {
	var str string
	if err := json.Unmarshal(b, &str); err != nil {
		return err
	}
	if str == "" {
		*s = SubscriptionID{}
		return nil
	}
	sub, err := NewSubscriptionID(str)
	if err != nil {
		return err
	}
	*s = sub
	return nil
}

// ---------------------------------------------------------------------------
// 2. Defined String Types & Enums
// ---------------------------------------------------------------------------

// ServiceID represents a system service identifier.
type ServiceID string

const (
	ServiceIDAICompletion ServiceID = "ai-completion"
	ServiceIDStorage      ServiceID = "storage"
	ServiceIDGeneral      ServiceID = "service1"
)

func (s ServiceID) String() string { return string(s) }
func (s ServiceID) IsZero() bool   { return s == "" }
func (s ServiceID) Validate() error {
	if s == "" {
		return errors.New("service_id cannot be empty")
	}
	return nil
}

// PlanID represents a subscription pricing tier.
type PlanID string

const (
	PlanIDFree       PlanID = "free"
	PlanIDPro        PlanID = "pro"
	PlanIDEnterprise PlanID = "enterprise"
)

func (p PlanID) String() string { return string(p) }
func (p PlanID) IsZero() bool   { return p == "" }
func (p PlanID) Validate() error {
	switch p {
	case PlanIDFree, PlanIDPro, PlanIDEnterprise:
		return nil
	default:
		if p == "" {
			return errors.New("plan_id cannot be empty")
		}
		return fmt.Errorf("unknown plan_id: %s", string(p))
	}
}

// QuotaLimit returns the default daily token quota for the plan tier.
func (p PlanID) QuotaLimit() int {
	switch p {
	case PlanIDPro:
		return 100000
	case PlanIDEnterprise:
		return 1000000
	case PlanIDFree:
		fallthrough
	default:
		return 1000
	}
}

// MetricID represents a billable resource metric.
type MetricID string

const (
	MetricIDTokens      MetricID = "tokens"
	MetricIDTotalTokens MetricID = "total_tokens"
	MetricIDRequests    MetricID = "requests"
)

func (m MetricID) String() string { return string(m) }
func (m MetricID) IsZero() bool   { return m == "" }
func (m MetricID) Validate() error {
	if m == "" {
		return errors.New("metric_id cannot be empty")
	}
	return nil
}

// EventTopic represents a topic key on the internal EventBus.
type EventTopic string

const (
	TopicUsageRecorded       EventTopic = "usage.recorded"
	TopicSubscriptionUpdated EventTopic = "subscription.updated"
	TopicKeyCreated          EventTopic = "key.created"
)

func (e EventTopic) String() string { return string(e) }
