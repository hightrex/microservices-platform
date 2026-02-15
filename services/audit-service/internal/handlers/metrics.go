package handlers

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	AuditLogsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "audit_logs_total",
			Help: "Total number of audit log entries",
		},
		[]string{"event_type", "outcome"},
	)

	AuditEventsConsumedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "audit_events_consumed_total",
			Help: "Total number of events consumed from other services",
		},
		[]string{"source_service"},
	)

	AuditExportDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "audit_export_duration_seconds",
			Help:    "Duration of audit export jobs in seconds",
			Buckets: prometheus.DefBuckets,
		},
	)

	AuditChainVerificationDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "audit_chain_verification_duration_seconds",
			Help:    "Duration of hash chain verification in seconds",
			Buckets: prometheus.DefBuckets,
		},
	)

	AuditRetentionCleanupTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "audit_retention_cleanup_total",
			Help: "Total number of audit logs cleaned up by retention policy",
		},
	)
)
