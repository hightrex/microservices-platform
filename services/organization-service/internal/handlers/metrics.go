package handlers

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// OrgMetrics holds Prometheus metrics specific to the organization service.
var OrgMetrics = struct {
	OrgCreatedTotal        prometheus.Counter
	ModuleToggledTotal     prometheus.Counter
	PlanChangedTotal       prometheus.Counter
	MemberInvitedTotal     prometheus.Counter
	MemberRemovedTotal     prometheus.Counter
}{
	OrgCreatedTotal: promauto.NewCounter(prometheus.CounterOpts{
		Name: "org_created_total",
		Help: "Total number of organizations created",
	}),
	ModuleToggledTotal: promauto.NewCounter(prometheus.CounterOpts{
		Name: "org_module_toggled_total",
		Help: "Total number of module toggle operations",
	}),
	PlanChangedTotal: promauto.NewCounter(prometheus.CounterOpts{
		Name: "org_plan_changed_total",
		Help: "Total number of plan change operations",
	}),
	MemberInvitedTotal: promauto.NewCounter(prometheus.CounterOpts{
		Name: "org_member_invited_total",
		Help: "Total number of member invitations",
	}),
	MemberRemovedTotal: promauto.NewCounter(prometheus.CounterOpts{
		Name: "org_member_removed_total",
		Help: "Total number of member removals",
	}),
}
