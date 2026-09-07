package api_v1

import (
	"net/http"
	"strconv"
	"time"

	appmetrics "agreements-generator/internal/metrics"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

func MWMetrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/metrics" || r.URL.Path == "/health" {
			next.ServeHTTP(w, r)
			return
		}

		start := time.Now()

		appmetrics.HTTPRequestsInFlight.Inc()
		defer appmetrics.HTTPRequestsInFlight.Dec()

		ww := chimiddleware.NewWrapResponseWriter(w, r.ProtoMajor)

		next.ServeHTTP(ww, r)

		route := chi.RouteContext(r.Context()).RoutePattern()
		if route == "" {
			route = "unknown"
		}

		status := ww.Status()
		if status == 0 {
			status = http.StatusOK
		}

		appmetrics.HTTPRequestsTotal.
			WithLabelValues(r.Method, route, strconv.Itoa(status)).
			Inc()

		appmetrics.HTTPRequestDuration.
			WithLabelValues(r.Method, route).
			Observe(time.Since(start).Seconds())
	})
}
