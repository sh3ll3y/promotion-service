package csv

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/sh3ll3y/promotion-service/internal/logging"
	"github.com/sh3ll3y/promotion-service/internal/models"
	"github.com/sh3ll3y/promotion-service/internal/types"
	"go.uber.org/zap"
)

type PromotionProcessor func(*models.Promotion) error

func ProcessPromotionsFromCSV(filename string, processor PromotionProcessor, workerCount int, eventPublisher types.EventPublisher) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)

	var wg sync.WaitGroup
	jobs := make(chan []string)
	errors := make(chan error, workerCount)

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go worker(&wg, jobs, errors, processor)
	}

	go func() {
		for {
			record, err := reader.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				errors <- fmt.Errorf("error reading CSV: %w", err)
				return
			}
			jobs <- record
		}
		close(jobs)
	}()

	go func() {
		wg.Wait()
		close(errors)
	}()

	for err := range errors {
		if err != nil {
			return err
		}
	}

	// Publish event after successful processing
	err = eventPublisher.PublishNewFileLoadedEvent()
	if err != nil {
		logging.Logger.Error("Failed to publish new file loaded event", zap.Error(err))
		return fmt.Errorf("failed to publish new file loaded event: %w", err)
	}

	return nil
}

func worker(wg *sync.WaitGroup, jobs <-chan []string, errors chan<- error, processor PromotionProcessor) {
	defer wg.Done()
	for record := range jobs {
		promotion, err := parsePromotion(record)
		if err != nil {
			errors <- err
			return
		}
		err = processor(promotion)
		if err != nil {
			errors <- err
			return
		}
	}
}

func parsePromotion(record []string) (*models.Promotion, error) {
	if len(record) != 3 {
		return nil, fmt.Errorf("invalid record length: got %d, want 3", len(record))
	}

	price, err := strconv.ParseFloat(record[1], 64)
	if err != nil {
		return nil, fmt.Errorf("invalid price: %w", err)
	}

	// Parse the date using the format from your CSV file
	expirationDate, err := time.Parse("2006-01-02 15:04:05 -0700 MST", record[2])
	if err != nil {
		return nil, fmt.Errorf("invalid expiration date: %w", err)
	}

	return &models.Promotion{
		ID:             record[0],
		Price:          price,
		ExpirationDate: expirationDate,
	}, nil
}