package billing

import "time"

// PricingModel defines the rating strategy for a metric (e.g., flat or per_unit).
type PricingModel string

const (
	PricingModelFlat    PricingModel = "flat"
	PricingModelPerUnit PricingModel = "per_unit"
)

// RateSpec defines pricing rules for a single metric.
// Pure primitive types (strings, float64, int64) — zero kernel dependencies.
type RateSpec struct {
	MetricID      string       `json:"metric_id"`
	PricingModel  PricingModel `json:"pricing_model"`
	UnitPrice     float64      `json:"unit_price"`     // Price per unit quantity
	UnitQuantity  int64        `json:"unit_quantity"`  // e.g., 1000 for $0.002 per 1,000 units
	IncludedQuota int64        `json:"included_quota"` // Quantity included free of charge
}

// PriceSchedule defines the pricing rules applied to calculate an invoice.
type PriceSchedule struct {
	ScheduleID string              `json:"schedule_id"`
	Currency   string              `json:"currency"`
	FlatFee    float64             `json:"flat_fee"`
	Rates      map[string]RateSpec `json:"rates"`
}

// MetricUsage holds aggregated consumption totals for a single metric.
type MetricUsage struct {
	MetricID   string `json:"metric_id"`
	Unit       string `json:"unit"`
	CycleTotal int64  `json:"cycle_total"`
}

// ServiceUsage holds usage data for a single service.
type ServiceUsage struct {
	ServiceID string                 `json:"service_id"`
	Metrics   map[string]MetricUsage `json:"metrics"`
}

// UsageOverview holds consolidated usage data across all services for a tenant.
type UsageOverview struct {
	TenantKey string         `json:"tenant_key"`
	Services  []ServiceUsage `json:"services"`
}

// InvoiceLineItem represents an itemized line item on an invoice.
type InvoiceLineItem struct {
	ServiceID   string  `json:"service_id"`
	MetricID    string  `json:"metric_id"`
	Description string  `json:"description"`
	Quantity    int64   `json:"quantity"`
	BillableQty int64   `json:"billable_quantity"`
	UnitPrice   float64 `json:"unit_price"`
	Amount      float64 `json:"amount"`
}

// Invoice represents an itemized financial billing invoice.
type Invoice struct {
	TenantKey   string            `json:"tenant_key"`
	Currency    string            `json:"currency"`
	LineItems   []InvoiceLineItem `json:"line_items"`
	Subtotal    float64           `json:"subtotal"`
	Tax         float64           `json:"tax"`
	Total       float64           `json:"total"`
	GeneratedAt time.Time         `json:"generated_at"`
}

// RatingEngine is a pure, thread-safe financial rating and invoice calculator.
type RatingEngine struct{}

// NewRatingEngine creates a new RatingEngine.
func NewRatingEngine() *RatingEngine {
	return &RatingEngine{}
}

// CalculateInvoice is a pure rating function converting usage data + price schedule into an invoice.
func (r *RatingEngine) CalculateInvoice(usage UsageOverview, schedule PriceSchedule) *Invoice {
	inv := &Invoice{
		TenantKey:   usage.TenantKey,
		Currency:    schedule.Currency,
		LineItems:   make([]InvoiceLineItem, 0),
		Subtotal:    schedule.FlatFee,
		GeneratedAt: time.Now().UTC(),
	}

	if schedule.FlatFee > 0 {
		inv.LineItems = append(inv.LineItems, InvoiceLineItem{
			Description: "Base Subscription Fee",
			Amount:      schedule.FlatFee,
		})
	}

	for _, service := range usage.Services {
		for metricID, metric := range service.Metrics {
			rate, ok := schedule.Rates[metricID]
			if !ok {
				continue
			}

			billableQty := metric.CycleTotal - rate.IncludedQuota
			if billableQty < 0 {
				billableQty = 0
			}

			unitQty := rate.UnitQuantity
			if unitQty <= 0 {
				unitQty = 1
			}

			amount := (float64(billableQty) / float64(unitQty)) * rate.UnitPrice
			inv.LineItems = append(inv.LineItems, InvoiceLineItem{
				ServiceID:   service.ServiceID,
				MetricID:    metricID,
				Description: service.ServiceID + " - " + metricID,
				Quantity:    metric.CycleTotal,
				BillableQty: billableQty,
				UnitPrice:   rate.UnitPrice,
				Amount:      amount,
			})
			inv.Subtotal += amount
		}
	}

	inv.Total = inv.Subtotal + inv.Tax
	return inv
}

// CalculateInvoice package-level convenience wrapper.
func CalculateInvoice(usage UsageOverview, schedule PriceSchedule) *Invoice {
	return NewRatingEngine().CalculateInvoice(usage, schedule)
}
