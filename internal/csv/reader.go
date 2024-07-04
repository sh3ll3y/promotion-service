package csv

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/sh3ll3y/promotion-service/internal/logging"
	"github.com/sh3ll3y/promotion-service/internal/metrics"
	"github.com/sh3ll3y/promotion-service/internal/models"
	"go.uber.org/zap"
)

type PromotionProcessor func(*models.Promotion) error

func ProcessPromotionsFromCSV(filename string, processor PromotionProcessor, numWorkers int) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			logging.Logger.Error("Failed to close file", zap.Error(closeErr))
		}
	}()

	reader := csv.NewReader(bufio.NewReader(file))

	var wg sync.WaitGroup
	recordChan := make(chan []string)
	errChan := make(chan error, numWorkers)

	// Start worker goroutines
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for record := range recordChan {
				promotion, err := parsePromotion(record)
				if err != nil {
					errChan <- fmt.Errorf("error parsing promotion: %w", err)
					return
				}

				err = processor(promotion)
				if err != nil {
					errChan <- fmt.Errorf("error processing promotion: %w", err)
					return
				}

				metrics.CsvProcessedLines.Inc()
			}
		}()
	}

	// Read CSV and send records to workers
	go func() {
		defer close(recordChan)
		for {
			record, err := reader.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				errChan <- fmt.Errorf("error reading CSV record: %w", err)
				return
			}
			recordChan <- record
		}
	}()

	// Wait for all workers to finish
	wg.Wait()
	close(errChan)

	// Check for any errors
	for err := range errChan {
		if err != nil {
			return err
		}
	}

	return nil
}

func parsePromotion(record []string) (*models.Promotion, error) {
	if len(record) != 3 {
		return nil, fmt.Errorf("invalid record format")
	}

	price, err := strconv.ParseFloat(record[1], 64)
	if err != nil {
		return nil, fmt.Errorf("invalid price format: %w", err)
	}

	expirationDate, err := time.Parse("2006-01-02 15:04:05 -0700 MST", record[2])
	if err != nil {
		return nil, fmt.Errorf("invalid date format: %w", err)
	}

	promotion := &models.Promotion{
		ID:             record[0],
		Price:          price,
		ExpirationDate: expirationDate,
	}

	if err := promotion.Validate(); err != nil {
		return nil, fmt.Errorf("invalid promotion data: %w", err)
	}

	return promotion, nil
}