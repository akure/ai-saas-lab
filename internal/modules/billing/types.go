package billing

import "aisaaslab/internal/kernel"

// Type aliases to expose kernel metering types through the billing package
type ChargeType = kernel.ChargeType

const (
	ChargeTypeMetered          = kernel.ChargeTypeMetered
	ChargeTypeRecurringMonthly = kernel.ChargeTypeRecurringMonthly
	ChargeTypeRecurringYearly  = kernel.ChargeTypeRecurringYearly
	ChargeTypeOneTime          = kernel.ChargeTypeOneTime
)

type ServiceSubscription = kernel.ServiceSubscription
type MeteringEvent = kernel.MeteringEvent
type MetricSummary = kernel.MetricSummary
type ServiceBillingStatement = kernel.ServiceBillingStatement
type TenantBillingOverview = kernel.TenantBillingOverview
