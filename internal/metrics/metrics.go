package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	HTTPRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "route", "status"},
	)

	HTTPRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "route"},
	)

	HTTPRequestsInFlight = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "http_requests_in_flight",
			Help: "Current number of HTTP requests being processed",
		},
	)

	JobsSubmittedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "generator_jobs_submitted_total",
			Help: "Total number of submitted generation jobs",
		},
		[]string{"mode", "result"},
	)

	JobSubmissionDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "generator_job_submission_duration_seconds",
			Help:    "Time spent accepting and dispatching a generation job",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"mode"},
	)

	JobsInProgress = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "generator_jobs_in_progress",
			Help: "Current number of generation jobs being processed",
		},
		[]string{"mode"},
	)

	JobProcessingDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "generator_job_processing_duration_seconds",
			Help:    "Time spent processing a generation job",
			Buckets: []float64{0.1, 0.25, 0.5, 1, 2, 5, 10, 20, 30, 60},
		},
		[]string{"mode", "result"},
	)
)

func init() {
	prometheus.MustRegister(
		HTTPRequestsTotal,
		HTTPRequestDuration,
		HTTPRequestsInFlight,
		JobsSubmittedTotal,
		JobSubmissionDuration,
		JobsInProgress,
		JobProcessingDuration,
	)
}
