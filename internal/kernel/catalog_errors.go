package kernel

import (
	"errors"
	"fmt"
)

// ---------------------------------------------------------------------------
// Catalog Sentinel Errors
// ---------------------------------------------------------------------------

// Sentinel errors for errors.Is comparisons across all catalog layers.
var (
	// ErrServiceAlreadyExists is returned when a tenant tries to register a
	// service_id that already exists. Maps to HTTP 409 Conflict.
	ErrServiceAlreadyExists = errors.New("service already exists")

	// ErrMetricAlreadyExists is returned when a metric_id already exists.
	// Maps to HTTP 409 Conflict.
	ErrMetricAlreadyExists = errors.New("metric already exists")

	// ErrPlanAlreadyExists is returned when a plan_id already exists.
	// Maps to HTTP 409 Conflict.
	ErrPlanAlreadyExists = errors.New("plan already exists")

	// ErrServiceNotFound is returned when a service_id lookup finds nothing.
	// Maps to HTTP 404 Not Found.
	ErrServiceNotFound = errors.New("service not found")

	// ErrMetricNotFound is returned when a metric_id lookup finds nothing.
	ErrMetricNotFound = errors.New("metric not found")

	// ErrPlanNotFound is returned when a plan_id lookup finds nothing.
	ErrPlanNotFound = errors.New("plan not found")

	// ErrCatalogValidation is the sentinel for any field-level validation
	// failure. Maps to HTTP 422 Unprocessable Entity.
	ErrCatalogValidation = errors.New("catalog validation error")

	// ErrCatalogBackendUnavailable is returned when all durable backends fail.
	// Maps to HTTP 503 Service Unavailable.
	ErrCatalogBackendUnavailable = errors.New("catalog backend unavailable")
)

// ---------------------------------------------------------------------------
// CatalogError — structured error carrying kind + context
// ---------------------------------------------------------------------------

// errorKind classifies a CatalogError for HTTP status mapping.
type errorKind int

const (
	kindConflict  errorKind = iota // 409
	kindNotFound                   // 404
	kindValidation                 // 422
	kindBackend                    // 503
)

// CatalogError is a structured error that wraps a sentinel Kind with additional
// context (field name, entity ID, backend name). It implements both error and
// Unwrap so errors.Is / errors.As work through the chain.
type CatalogError struct {
	Kind    error     // sentinel: ErrServiceAlreadyExists, ErrCatalogValidation, …
	kind    errorKind // internal classification for HTTP mapping
	Message string    // human-readable detail
	Backend string    // backend name ("redis", "postgres") if applicable
	Wrapped error     // optional inner error
}

func (e *CatalogError) Error() string {
	if e.Wrapped != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Wrapped)
	}
	return e.Message
}

// Unwrap supports errors.Is / errors.As on the chain:
//   - errors.Is(err, ErrServiceAlreadyExists) works through CatalogError
func (e *CatalogError) Unwrap() []error {
	if e.Wrapped != nil {
		return []error{e.Kind, e.Wrapped}
	}
	return []error{e.Kind}
}

// ---------------------------------------------------------------------------
// Constructors
// ---------------------------------------------------------------------------

// NewConflictError returns a 409-class CatalogError for a duplicate entity.
// entityType should be "service", "metric", or "plan".
// id is the conflicting identifier string.
func NewConflictError(entityType, id string) *CatalogError {
	var sentinel error
	switch entityType {
	case "metric":
		sentinel = ErrMetricAlreadyExists
	case "plan":
		sentinel = ErrPlanAlreadyExists
	default:
		sentinel = ErrServiceAlreadyExists
	}
	return &CatalogError{
		Kind:    sentinel,
		kind:    kindConflict,
		Message: fmt.Sprintf("%s %q already exists", entityType, id),
	}
}

// NewValidationError returns a 422-class CatalogError for field-level failures.
// field is the field name (e.g. "service_id"), reason is a human description.
func NewValidationError(field, reason string) *CatalogError {
	return &CatalogError{
		Kind:    ErrCatalogValidation,
		kind:    kindValidation,
		Message: fmt.Sprintf("validation failed on %q: %s", field, reason),
	}
}

// NewNotFoundError returns a 404-class CatalogError for missing entities.
func NewNotFoundError(entityType, id string) *CatalogError {
	var sentinel error
	switch entityType {
	case "metric":
		sentinel = ErrMetricNotFound
	case "plan":
		sentinel = ErrPlanNotFound
	default:
		sentinel = ErrServiceNotFound
	}
	return &CatalogError{
		Kind:    sentinel,
		kind:    kindNotFound,
		Message: fmt.Sprintf("%s %q not found", entityType, id),
	}
}

// NewBackendError returns a 503-class CatalogError for durable backend failures.
// backend is the backend name (e.g. "postgres"); inner is the originating error.
func NewBackendError(backend string, inner error) *CatalogError {
	return &CatalogError{
		Kind:    ErrCatalogBackendUnavailable,
		kind:    kindBackend,
		Message: fmt.Sprintf("catalog backend %q unavailable", backend),
		Backend: backend,
		Wrapped: inner,
	}
}

// ---------------------------------------------------------------------------
// HTTP classification helpers (used by handlers layer)
// ---------------------------------------------------------------------------

// IsCatalogConflict reports whether err represents a duplicate-entity conflict.
// Maps to HTTP 409.
func IsCatalogConflict(err error) bool {
	var ce *CatalogError
	if errors.As(err, &ce) {
		return ce.kind == kindConflict
	}
	return errors.Is(err, ErrServiceAlreadyExists) ||
		errors.Is(err, ErrMetricAlreadyExists) ||
		errors.Is(err, ErrPlanAlreadyExists)
}

// IsCatalogNotFound reports whether err is a not-found error.
// Maps to HTTP 404.
func IsCatalogNotFound(err error) bool {
	var ce *CatalogError
	if errors.As(err, &ce) {
		return ce.kind == kindNotFound
	}
	return errors.Is(err, ErrServiceNotFound) ||
		errors.Is(err, ErrMetricNotFound) ||
		errors.Is(err, ErrPlanNotFound)
}

// IsCatalogValidation reports whether err is a field-validation failure.
// Maps to HTTP 422.
func IsCatalogValidation(err error) bool {
	var ce *CatalogError
	if errors.As(err, &ce) {
		return ce.kind == kindValidation
	}
	return errors.Is(err, ErrCatalogValidation)
}

// IsCatalogBackend reports whether err is a durable-backend failure.
// Maps to HTTP 503.
func IsCatalogBackend(err error) bool {
	var ce *CatalogError
	if errors.As(err, &ce) {
		return ce.kind == kindBackend
	}
	return errors.Is(err, ErrCatalogBackendUnavailable)
}
