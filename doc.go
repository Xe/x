// Package x is a hack
package x

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	Version = "devel"

	gauge *prometheus.GaugeVec
)

func init() {
	gauge = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "within_website_x_version",
		Help: "The version of within.website/x in use.",
	}, []string{"version"})

	gauge.WithLabelValues(Version).Inc()
}
