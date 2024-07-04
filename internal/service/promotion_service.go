package service

import (
	"fmt"
	"github.com/sh3ll3y/promotion-service/internal/csv"
	"github.com/sh3ll3y/promotion-service/internal/logging"
	"github.com/sh3ll3y/promotion-service/internal/models"
	"github.com/sh3ll3y/promotion-service/internal/repository"
	"github.com/sh3ll3y/promotion-service/internal/types"
	"go.uber.org/zap"
	"sync"
)

type PromotionService struct {
	writeRepo     *repository.WriteRepository
	readRepo      *repository.ReadRepository
	eventPublisher types.EventPublisher
}

func NewPromotionService(writeRepo *repository.WriteRepository, readRepo *repository.ReadRepository, eventPublisher types.EventPublisher) *PromotionService {
	return &PromotionService{
		writeRepo:     writeRepo,
		readRepo:      readRepo,
		eventPublisher: eventPublisher,
	}
}

func (s *PromotionService) ProcessCSVFile(filename string) error {
	logging.Logger.Info("Starting CSV processing", zap.String("filename", filename))

	// Clear write DB
	err := s.writeRepo.ClearAllPromotions()
	if err != nil {
		return fmt.Errorf("failed to clear promotions in write DB: %w", err)
	}

	// Read and process CSV
	err = csv.ProcessPromotionsFromCSV(filename, s.writeRepo.CreatePromotion, 5, s.eventPublisher)
	if err != nil {
		return fmt.Errorf("failed to process CSV: %w", err)
	}

	logging.Logger.Info("CSV processing completed successfully")
	return nil
}

func (s *PromotionService) UpdateReadDB() error {
	logging.Logger.Info("Starting read DB update")

	// Clear temp table
	err := s.readRepo.ClearTempTable()
	if err != nil {
		return fmt.Errorf("failed to clear temp table: %w", err)
	}

	// Get total count
	totalCount, err := s.writeRepo.GetTotalPromotionsCount()
	if err != nil {
		return fmt.Errorf("failed to get total promotions count: %w", err)
	}

	// Set up worker pool
	workerCount := 10
	batchSize := 1000
	var wg sync.WaitGroup
	errChan := make(chan error, workerCount)

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for offset := workerID * batchSize; offset < totalCount; offset += workerCount * batchSize {
				err := s.processPromotionsBatch(offset, batchSize)
				if err != nil {
					errChan <- err
					return
				}
			}
		}(i)
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		if err != nil {
			return fmt.Errorf("error in worker: %w", err)
		}
	}

	// Swap tables
	err = s.readRepo.SwapTables()
	if err != nil {
		return fmt.Errorf("failed to swap tables: %w", err)
	}

	logging.Logger.Info("Read DB update completed successfully")
	return nil
}

func (s *PromotionService) processPromotionsBatch(offset, limit int) error {
	promotions, err := s.writeRepo.GetPromotionsBatch(offset, limit)
	if err != nil {
		return fmt.Errorf("failed to get promotions batch: %w", err)
	}

	err = s.readRepo.BulkInsertPromotions(promotions)
	if err != nil {
		return fmt.Errorf("failed to insert promotions batch: %w", err)
	}

	return nil
}

func (s *PromotionService) GetPromotion(id string) (*models.Promotion, error) {
	promotion, err := s.readRepo.GetPromotion(id)
	if err != nil {
		logging.Logger.Error("Failed to get promotion", zap.Error(err), zap.String("id", id))
		return nil, err
	}
	return promotion, nil
}