package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	namespace = "nais"
	subsystem = "gemini"
)

const (
	LabelStatus = "status"
	StatusOK    = "ok"
	StatusError = "error"
)

var (
	databaseQueries = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:      "database_queries",
		Help:      "time to execute database queries",
		Namespace: namespace,
		Subsystem: subsystem,
		Buckets:   prometheus.LinearBuckets(0.005, 0.005, 20),
	},
		[]string{
			LabelStatus,
		},
	)
)

func statusLabel(err error) string {
	if err == nil {
		return StatusOK
	}
	return StatusError
}

func DatabaseQuery(t time.Time, err error) {
	elapsed := time.Since(t)
	databaseQueries.With(prometheus.Labels{
		LabelStatus: statusLabel(err),
	}).Observe(elapsed.Seconds())
}
