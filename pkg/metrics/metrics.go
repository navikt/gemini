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

	LabelCalendar = "calendar"
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

	synchronizations = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:      "synchronizations",
			Help:      "time to synchronize calendars",
			Namespace: namespace,
			Subsystem: subsystem,
			Buckets:   prometheus.LinearBuckets(0.0, 5.0, 24), // five-second intervals up to two minutes
		},
		[]string{
			LabelStatus,
		},
	)

	requests = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:      "requests",
			Help:      "calendar requests",
			Namespace: namespace,
			Subsystem: subsystem,
		},
		[]string{
			LabelStatus,
			LabelCalendar,
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
	prometheus.MustRegister(requests)
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

func Synchronizations(t time.Time, err error) {
	elapsed := time.Since(t)
	synchronizations.With(prometheus.Labels{
		LabelStatus: statusLabel(err),
	}).Observe(elapsed.Seconds())
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

func Request(t time.Time, calendarName string, success bool) {
	duration := time.Since(t)
	if success {
		requests.WithLabelValues(StatusOK, calendarName).Observe(duration.Seconds())
	} else {
		requests.WithLabelValues(StatusError, calendarName).Observe(duration.Seconds())
	}
}
