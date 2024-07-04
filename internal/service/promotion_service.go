package service

import (
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/sh3ll3y/promotion-service/internal/csv"
	"github.com/sh3ll3y/promotion-service/internal/kafka"
	"github.com/sh3ll3y/promotion-service/internal/metrics"
	"github.com/sh3ll3y/promotion-service/internal/models"
	"github.com/sh3ll3y/promotion-service/internal/repository"
)

type PromotionService struct {
	writeRepo     *repository.WriteRepository
	readRepo      *repository.ReadRepository
	cache         *redis.Client
	kafkaProducer *kafka.Producer
}

func NewPromotionService(writeRepo *repository.WriteRepository, readRepo *repository.ReadRepository, cache *redis.Client, kafkaProducer *kafka.Producer) *PromotionService {
	return &PromotionService{
		writeRepo:     writeRepo,
		readRepo:      readRepo,
		cache:         cache,
		kafkaProducer: kafkaProducer,
	}
}

func (s *PromotionService) CreatePromotion(p *models.Promotion) error {
	// Start a database transaction
	tx, err := s.writeRepo.BeginTx()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback() // Rollback if not committed

	// Write to database within the transaction
	err = s.writeRepo.CreatePromotionTx(tx, p)
	if err != nil {
		return fmt.Errorf("failed to create promotion: %w", err)
	}

	// Publish to Kafka
	err = s.kafkaProducer.PublishPromotion(p)
	if err != nil {
		return fmt.Errorf("failed to publish promotion to Kafka: %w", err)
	}

	// If we got here, both database write and Kafka publish succeeded
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	metrics.KafkaPublishedMessages.Inc()
	return nil
}

func (s *PromotionService) GetPromotion(id string) (*models.Promotion, error) {
	return s.readRepo.GetPromotion(id)
}

func (s *PromotionService) ProcessCSVFile(filename string) error {
	return csv.ProcessPromotionsFromCSV(filename, s.processPromotion, 5)
}

func (s *PromotionService) processPromotion(p *models.Promotion) error {
	// Start a database transaction
	tx, err := s.writeRepo.BeginTx()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}

	defer func() {
		if r := recover(); r != interface{}(nil) {
			tx.Rollback()
			panic(r) // re-throw panic after Rollback
		} else if err != nil {
			tx.Rollback() // err is non-nil; don't change it
		} else {
			err = tx.Commit() // err is nil; if Commit returns error update err
		}
	}()

	// Write to database within the transaction
	err = s.writeRepo.CreatePromotionTx(tx, p)
	if err != nil {
		return fmt.Errorf("failed to create promotion: %w", err)
	}

	// Publish to Kafka
	err = s.kafkaProducer.PublishPromotion(p)
	if err != nil {
		return fmt.Errorf("failed to publish promotion to Kafka: %w", err)
	}

	metrics.KafkaPublishedMessages.Inc()
	return nil
}