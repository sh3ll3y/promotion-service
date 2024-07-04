package main

import (
	"context"
	"database/sql"
	"github.com/sh3ll3y/promotion-service/internal/types"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sh3ll3y/promotion-service/internal/api"
	"github.com/sh3ll3y/promotion-service/internal/config"
	"github.com/sh3ll3y/promotion-service/internal/database"
	"github.com/sh3ll3y/promotion-service/internal/kafka"
	"github.com/sh3ll3y/promotion-service/internal/logging"
	"github.com/sh3ll3y/promotion-service/internal/repository"
	"github.com/sh3ll3y/promotion-service/internal/service"
	"go.uber.org/zap"
)

func main() {
	logging.Init()
	defer logging.Logger.Sync()

	cfg, err := config.Load()
	if err != nil {
		logging.Logger.Fatal("Failed to load config", zap.Error(err))
	}

	// Open database connections
	writeDB, err := connectWithRetry(cfg.WriteDBURL)
	if err != nil {
		logging.Logger.Fatal("Failed to connect to write database", zap.Error(err))
	}
	defer writeDB.Close()

	readDB, err := connectWithRetry(cfg.ReadDBURL)
	if err != nil {
		logging.Logger.Fatal("Failed to connect to read database", zap.Error(err))
	}
	defer readDB.Close()

	// Run migrations
	if err := database.RunMigrations(writeDB, "./migrations"); err != nil {
		logging.Logger.Fatal("Failed to run migrations on write database", zap.Error(err))
	}
	if err := database.RunMigrations(readDB, "./migrations"); err != nil {
		logging.Logger.Fatal("Failed to run migrations on read database", zap.Error(err))
	}

	writeRepo := repository.NewWriteRepository(writeDB)

	redisOpts, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		logging.Logger.Fatal("Failed to parse Redis URL", zap.Error(err))
	}

	cacheClient := redis.NewClient(redisOpts)

	// Ping the Redis server to check the connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := cacheClient.Ping(ctx).Err(); err != nil {
		logging.Logger.Fatal("Failed to connect to Redis", zap.Error(err))
	}

	readRepo := repository.NewReadRepository(readDB, cacheClient)

	kafkaProducer, err := kafka.NewProducer(cfg.KafkaBrokers, cfg.KafkaTopic)
	if err != nil {
		logging.Logger.Fatal("Failed to create Kafka producer", zap.Error(err))
	}
	var eventPublisher types.EventPublisher = kafkaProducer
	promotionService := service.NewPromotionService(writeRepo, readRepo, eventPublisher)

	kafkaConsumer, err := kafka.NewConsumer(cfg.KafkaBrokers, cfg.KafkaTopic, promotionService)
	if err != nil {
		logging.Logger.Fatal("Failed to create Kafka consumer", zap.Error(err))
	}

	go func() {
		if err := kafkaConsumer.Start(); err != nil {
			logging.Logger.Error("Kafka consumer error", zap.Error(err))
		}
	}()

	router := mux.NewRouter()
	api.RegisterHandlers(router, promotionService)
	router.Handle("/metrics", promhttp.Handler())

	srv := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	go func() {
		logging.Logger.Info("Starting server", zap.String("address", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logging.Logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logging.Logger.Info("Shutting down server...")

	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logging.Logger.Fatal("Server forced to shutdown", zap.Error(err))
	}

	logging.Logger.Info("Server exiting")
}

func connectWithRetry(dbURL string) (*sql.DB, error) {
	var db *sql.DB
	var err error
	for i := 0; i < 5; i++ {
		db, err = sql.Open("postgres", dbURL)
		if err == nil {
			if err = db.Ping(); err == nil {
				return db, nil
			}
		}
		logging.Logger.Warn("Failed to connect to database, retrying...", zap.Error(err), zap.Int("attempt", i+1))
		time.Sleep(5 * time.Second)
	}
	return nil, err
}