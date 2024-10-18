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
	LabelActive = "active"
	ActiveYes   = "true"
	ActiveNo    = "false"

	LabelStatus = "status"
	StatusOK    = "ok"
	StatusError = "error"
)

var (
	users = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name:      "users",
			Help:      "number of managed users",
			Namespace: namespace,
			Subsystem: subsystem,
		},
		[]string{
			LabelActive,
		},
	)

	queueSize = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name:      "queue_size",
			Help:      "number of calendar synchronizations in queue",
			Namespace: namespace,
			Subsystem: subsystem,
		},
	)

	synchronizations = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name:      "synchronizations",
			Help:      "number of managed users",
			Namespace: namespace,
			Subsystem: subsystem,
		},
		[]string{
			LabelStatus,
		},
	)

	databaseQueries = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
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

func init() {
	prometheus.MustRegister(users)
	prometheus.MustRegister(queueSize)
	prometheus.MustRegister(synchronizations)
	prometheus.MustRegister(databaseQueries)

	synchronizations.With(prometheus.Labels{
		LabelStatus: StatusOK,
	})
	synchronizations.With(prometheus.Labels{
		LabelStatus: StatusError,
	})
}

func statusLabel(err error) string {
	if err == nil {
		return StatusOK
	}
	return StatusError
}

func Synchronizations(err error) {
	synchronizations.With(prometheus.Labels{
		LabelStatus: statusLabel(err),
	}).Inc()
}

func Users(active, inactive int) {
	users.With(prometheus.Labels{
		LabelActive: ActiveYes,
	}).Set(float64(active))

	users.With(prometheus.Labels{
		LabelActive: ActiveNo,
	}).Set(float64(inactive))
}

func DatabaseQuery(t time.Time, err error) {
	elapsed := time.Since(t)
	databaseQueries.With(prometheus.Labels{
		LabelStatus: statusLabel(err),
	}).Observe(elapsed.Seconds())
}

func QueueSize(length int) {
	queueSize.Set(float64(length))
}
