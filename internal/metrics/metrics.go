package metrics

import (
	"log/slog"
	"net/http"

	grpcprom "github.com/grpc-ecosystem/go-grpc-middleware/providers/prometheus"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
)

// Metrics wraps the prometheus metrics gathering and serving.
//
// It exposes gRPC and Go Runtime metrics.
type Metrics struct {
	logger      *slog.Logger
	addr        string
	reg         *prometheus.Registry
	grpcMetrics *grpcprom.ServerMetrics
	goMetrics   prometheus.Collector
}

func New(logger *slog.Logger, addr string) *Metrics {
	metrics := &Metrics{
		logger: logger,
		addr:   addr,
		reg:    prometheus.NewRegistry(),
		grpcMetrics: grpcprom.NewServerMetrics(
			grpcprom.WithServerHandlingTimeHistogram(),
		),
		goMetrics: collectors.NewGoCollector(),
	}

	metrics.logger.Debug(
		"registering metrics with registry",
	)

	metrics.reg.MustRegister(metrics.goMetrics)
	metrics.reg.MustRegister(metrics.grpcMetrics)

	metrics.logger.Debug(
		"registered metrics",
	)

	return metrics
}

// UnaryServerInterceptor returns an UnaryServerInterceptor that must be
// installed on the grpc.Server to gather metrics.
func (s *Metrics) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return s.grpcMetrics.UnaryServerInterceptor()
}

// InitializeMetrics initializes all metrics, with their appropriate null value,
// for all gRPC methods registered on a gRPC server.
// This is useful, to ensure that all metrics exist when collecting and querying.
func (s *Metrics) InitializeMetrics(server *grpc.Server) {
	s.grpcMetrics.InitializeMetrics(server)
}

func (s *Metrics) Registry() *prometheus.Registry {
	return s.reg
}
func (s *Metrics) Serve() {
	httpServer := &http.Server{Handler: promhttp.HandlerFor(s.reg, promhttp.HandlerOpts{}), Addr: s.addr}

	s.logger.Debug(
		"starting prometheus http server",
		"addr", s.addr,
	)

	go func() {
		if err := httpServer.ListenAndServe(); err != nil {
			s.logger.Error(
				"Unable to start the prometheus http server",
				"err", err,
			)
		}
	}()
}
