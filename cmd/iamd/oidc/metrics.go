package oidc

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	refreshReuse           = promauto.NewCounter(prometheus.CounterOpts{Name: "iamd_oidc_refresh_reuse_total", Help: "Detected replay of a rotated refresh token."})
	registrationRejections = promauto.NewCounterVec(prometheus.CounterOpts{Name: "iamd_oidc_registration_rejections_total", Help: "Rejected dynamic client registrations by OAuth error."}, []string{"error"})
	metadataFailures       = promauto.NewCounter(prometheus.CounterOpts{Name: "iamd_oidc_metadata_failures_total", Help: "Failed URL client metadata resolution or validation."})
)

func registrationProblem(w http.ResponseWriter, code string, status int) {
	registrationRejections.WithLabelValues(code).Inc()
	problem(w, code, status)
}
