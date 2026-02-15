package handlers

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// NotificationsSentTotal tracks total notifications sent by channel and status.
	NotificationsSentTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "notifications_sent_total",
			Help: "Total number of notifications sent",
		},
		[]string{"channel", "status"},
	)

	// NotificationsDeliveryDuration tracks delivery duration by channel.
	NotificationsDeliveryDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "notifications_delivery_duration_seconds",
			Help:    "Notification delivery duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"channel"},
	)

	// NotificationsRetryTotal tracks retry attempts by channel.
	NotificationsRetryTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "notifications_retry_total",
			Help: "Total number of notification retry attempts",
		},
		[]string{"channel"},
	)

	// NotificationsDLQTotal tracks messages sent to DLQ.
	NotificationsDLQTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "notifications_dlq_total",
			Help: "Total number of notifications moved to DLQ",
		},
	)
)
