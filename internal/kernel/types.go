package kernel

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// serviceIDPattern allows lowercase alphanumeric, hyphens, and underscores.
// Must start and end with an alphanumeric character. Max 64 chars.
var serviceIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,62}[a-z0-9]$|^[a-z0-9]$`)

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
func (t TenantKey) IsZero() bool   { return t.value == "" }

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
func (s SubscriptionID) IsZero() bool   { return s.value == "" }

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
// 2. Defined String Types & Enums (strongly typed strings)
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
	if len(s) > 64 {
		return fmt.Errorf("service_id %q exceeds maximum length of 64 characters", string(s))
	}
	if !serviceIDPattern.MatchString(string(s)) {
		return fmt.Errorf("service_id %q is invalid: must be lowercase alphanumeric with hyphens/underscores, no leading/trailing hyphens", string(s))
	}
	return nil
}

// MaaSPlanID represents a MaaS platform infrastructure subscription tier.
type MaaSPlanID string

const (
	MaaSPlanStarter    MaaSPlanID = "maas-starter"
	MaaSPlanGrowth     MaaSPlanID = "maas-growth"
	MaaSPlanEnterprise MaaSPlanID = "maas-enterprise"

	// Backward-compatibility aliases during refactoring transition
	PlanIDFree       MaaSPlanID = "free"
	PlanIDPro        MaaSPlanID = "pro"
	PlanIDEnterprise MaaSPlanID = "enterprise"
)

// Backward-compatibility type alias
type PlanID = MaaSPlanID

func (p MaaSPlanID) String() string { return string(p) }
func (p MaaSPlanID) IsZero() bool   { return p == "" }
func (p MaaSPlanID) Validate() error {
	switch p {
	case MaaSPlanStarter, MaaSPlanGrowth, MaaSPlanEnterprise, PlanIDFree, PlanIDPro, PlanIDEnterprise:
		return nil
	default:
		if p == "" {
			return errors.New("maas_plan_id cannot be empty")
		}
		return fmt.Errorf("unknown maas_plan_id: %s", string(p))
	}
}

// TODO - To be changed after POC and before MVP
// QuotaLimit returns the default daily token ingestion limit for the MaaS platform tier.
func (p MaaSPlanID) QuotaLimit() int {
	switch p {
	case MaaSPlanGrowth, PlanIDPro:
		return 100000
	case MaaSPlanEnterprise, PlanIDEnterprise:
		return 1000000
	case MaaSPlanStarter, PlanIDFree:
		fallthrough
	default:
		return 1000
	}
}

// ApplicationPlanID represents a dynamic end-customer plan identifier defined by a Tenant.
type ApplicationPlanID string

func (a ApplicationPlanID) String() string { return string(a) }
func (a ApplicationPlanID) IsZero() bool   { return a == "" }

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
	if len(m) > 64 {
		return fmt.Errorf("metric_id %q exceeds maximum length of 64 characters", string(m))
	}
	return nil
}

// ---------------------------------------------------------------------------
// 3. Tenant Self-Service Catalog Descriptors
// ---------------------------------------------------------------------------

// TenantServiceDescriptor defines a dynamic service registered by a Tenant.
type TenantServiceDescriptor struct {
	ServiceID   ServiceID `json:"service_id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	CreatedAt   string    `json:"created_at,omitempty"`
}

// Validate checks all fields of TenantServiceDescriptor for correctness.
// Returns a *CatalogError wrapping ErrCatalogValidation on any failure.
func (d TenantServiceDescriptor) Validate() error {
	if err := d.ServiceID.Validate(); err != nil {
		return NewValidationError("service_id", err.Error())
	}
	name := strings.TrimSpace(d.Name)
	if name == "" {
		return NewValidationError("name", "name cannot be empty")
	}
	if len(name) > 128 {
		return NewValidationError("name", fmt.Sprintf("name exceeds maximum length of 128 characters (got %d)", len(name)))
	}
	if len(d.Description) > 512 {
		return NewValidationError("description", fmt.Sprintf("description exceeds maximum length of 512 characters (got %d)", len(d.Description)))
	}
	return nil
}

// TenantMetricDescriptor defines a dynamic billable metric registered under a Tenant Service.
type TenantMetricDescriptor struct {
	MetricID    MetricID  `json:"metric_id"`
	ServiceID   ServiceID `json:"service_id"`
	Name        string    `json:"name"`
	Unit        string    `json:"unit"` // e.g. "tokens", "seconds", "bytes", "requests"
	Description string    `json:"description,omitempty"`
	CreatedAt   string    `json:"created_at,omitempty"`
}

// Validate checks all fields of TenantMetricDescriptor.
func (d TenantMetricDescriptor) Validate() error {
	if err := d.MetricID.Validate(); err != nil {
		return NewValidationError("metric_id", err.Error())
	}
	if err := d.ServiceID.Validate(); err != nil {
		return NewValidationError("service_id", err.Error())
	}
	if len(strings.TrimSpace(d.Name)) > 128 {
		return NewValidationError("name", "name exceeds maximum length of 128 characters")
	}
	if len(d.Description) > 512 {
		return NewValidationError("description", "description exceeds maximum length of 512 characters")
	}
	unit := strings.TrimSpace(d.Unit)
	if unit == "" {
		return NewValidationError("unit", "unit cannot be empty (e.g. \"tokens\", \"requests\", \"bytes\")")
	}
	return nil
}

// TenantPlanDescriptor defines a dynamic pricing tier created by a Tenant for their end customers.
type TenantPlanDescriptor struct {
	PlanID         ApplicationPlanID    `json:"plan_id"`
	ServiceID      ServiceID            `json:"service_id"`
	Name           string               `json:"name"`
	Rates          map[MetricID]float64 `json:"rates"`           // Cost per unit (e.g. "tokens": 0.002)
	IncludedQuotas map[MetricID]int64   `json:"included_quotas"` // Included free usage per cycle
	Version        int                  `json:"version"`
	Active         bool                 `json:"active"`
	CreatedAt      string               `json:"created_at,omitempty"`
}

// EventTopic represents a topic key on the internal EventBus.
type EventTopic string

const (
	// Metering & subscription topics
	TopicUsageRecorded       EventTopic = "usage.recorded"
	TopicSubscriptionUpdated EventTopic = "subscription.updated"
	TopicKeyCreated          EventTopic = "key.created"

	// Tenant catalog topics — published by tenant.Service, consumed by any subscriber.
	TopicServiceRegistered EventTopic = "tenant.service.registered"
	TopicMetricRegistered  EventTopic = "tenant.metric.registered"
	TopicPlanRegistered    EventTopic = "tenant.plan.created"
)

func (e EventTopic) String() string { return string(e) }

// ---------------------------------------------------------------------------
// Typed Event Payload Structs (catalog)
// ---------------------------------------------------------------------------

// ServiceRegisteredEvent is published on TopicServiceRegistered when a tenant
// successfully registers a new service descriptor.
type ServiceRegisteredEvent struct {
	TenantKey    string    `json:"tenant_key"`
	ServiceID    string    `json:"service_id"`
	Name         string    `json:"name"`
	RegisteredAt time.Time `json:"registered_at"`
}

// MetricRegisteredEvent is published on TopicMetricRegistered.
type MetricRegisteredEvent struct {
	TenantKey    string    `json:"tenant_key"`
	MetricID     string    `json:"metric_id"`
	ServiceID    string    `json:"service_id"`
	Unit         string    `json:"unit"`
	RegisteredAt time.Time `json:"registered_at"`
}

// PlanRegisteredEvent is published on TopicPlanRegistered.
type PlanRegisteredEvent struct {
	TenantKey    string    `json:"tenant_key"`
	PlanID       string    `json:"plan_id"`
	ServiceID    string    `json:"service_id"`
	Version      int       `json:"version"`
	RegisteredAt time.Time `json:"registered_at"`
}
