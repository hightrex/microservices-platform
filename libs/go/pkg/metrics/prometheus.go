package metrics

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	httpDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "http_request_duration_seconds",
		Help: "Duration of HTTP requests.",
	}, []string{"method", "path", "status"})

	// EventProcessingDuration tracks the time taken to process events
	EventProcessingDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "event_processing_duration_seconds",
		Help: "Duration of event processing.",
	}, []string{"stream", "event_type", "status"})

	// EventsProcessed tracks the total number of events processed
	EventsProcessed = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "events_processed_total",
		Help: "Total number of events processed.",
	}, []string{"stream", "event_type", "status"})
)

// Middleware returns a Gin middleware for collecting HTTP metrics
func Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		duration := time.Since(start).Seconds()
		status := strconv.Itoa(c.Writer.Status())
		httpDuration.WithLabelValues(c.Request.Method, c.Request.URL.Path, status).Observe(duration)
	}
}
