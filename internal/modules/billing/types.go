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
type ServiceUsageStatement = kernel.ServiceUsageStatement
type TenantUsageOverview = kernel.TenantUsageOverview
type ServiceBillingStatement = kernel.ServiceUsageStatement
type TenantBillingOverview = kernel.TenantUsageOverview

// ToUsageOverview converts a kernel.TenantUsageOverview into an extract-ready billing.UsageOverview.
func ToUsageOverview(overview kernel.TenantUsageOverview) UsageOverview {
	res := UsageOverview{
		TenantKey: overview.TenantKey.String(),
		Services:  make([]ServiceUsage, 0, len(overview.Statements)),
	}

	for _, stmt := range overview.Statements {
		sUsage := ServiceUsage{
			ServiceID: stmt.ServiceID.String(),
			Metrics:   make(map[string]MetricUsage, len(stmt.Metrics)),
		}
		for mID, summary := range stmt.Metrics {
			if summary == nil {
				continue
			}
			sUsage.Metrics[mID.String()] = MetricUsage{
				MetricID:   summary.MetricID.String(),
				Unit:       summary.Unit,
				CycleTotal: summary.CycleTotal,
			}
		}
		res.Services = append(res.Services, sUsage)
	}
	return res
}
