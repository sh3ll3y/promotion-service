package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	CsvProcessedLines = promauto.NewCounter(prometheus.CounterOpts{
		Name: "csv_processed_lines_total",
		Help: "The total number of processed lines from CSV",
	})

	DatabaseOperations = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "database_operations_total",
		Help: "The total number of database operations",
	}, []string{"operation"})

	CacheHits = promauto.NewCounter(prometheus.CounterOpts{
		Name: "cache_hits_total",
		Help: "The total number of cache hits",
	})

	CacheMisses = promauto.NewCounter(prometheus.CounterOpts{
		Name: "cache_misses_total",
		Help: "The total number of cache misses",
	})

	KafkaPublishedMessages = promauto.NewCounter(prometheus.CounterOpts{
		Name: "kafka_published_messages_total",
		Help: "The total number of messages published to Kafka",
	})

	ApiRequests = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "api_requests_total",
		Help: "The total number of API requests",
	}, []string{"endpoint", "method", "status"})

	ApiLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "api_request_duration_seconds",
		Help:    "The latency of API requests",
		Buckets: prometheus.DefBuckets,
	}, []string{"endpoint", "method"})
)