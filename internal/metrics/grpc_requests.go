package metrics

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var grpcRequestsTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "grpc_requests_total",
		Help: "Total number of gRPC requests handled by the service.",
	},
	[]string{"service", "method", "status"},
)

var grpcRequestDurationSeconds = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "grpc_request_duration_seconds",
		Help:    "gRPC request latency in seconds.",
		Buckets: prometheus.DefBuckets,
	},
	[]string{"service", "method", "status"},
)

func init() {
	prometheus.MustRegister(grpcRequestsTotal)
	prometheus.MustRegister(grpcRequestDurationSeconds)
}

func GRPCMetricsUnaryInterceptor(service string) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		start := time.Now()
		resp, err := handler(ctx, req)
		grpcStatus := status.Code(err)
		if err == nil {
			grpcStatus = codes.OK
		}
		statusLabel := grpcStatus.String()

		grpcRequestsTotal.WithLabelValues(service, info.FullMethod, statusLabel).Inc()
		grpcRequestDurationSeconds.WithLabelValues(service, info.FullMethod, statusLabel).
			Observe(time.Since(start).Seconds())

		return resp, err
	}
}
