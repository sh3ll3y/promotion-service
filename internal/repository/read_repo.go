package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/sh3ll3y/promotion-service/internal/logging"
	"github.com/sh3ll3y/promotion-service/internal/metrics"
	"github.com/sh3ll3y/promotion-service/internal/models"
	"go.uber.org/zap"
	"strings"
	"time"
)

type ReadRepository struct {
	db    *sql.DB
	cache *redis.Client
}

func NewReadRepository(db *sql.DB, cache *redis.Client) *ReadRepository {
	return &ReadRepository{db: db, cache: cache}
}

func (r *ReadRepository) ClearTempTable() error {
	_, err := r.db.Exec("DELETE FROM promotions_temp")
	return err
}

func (r *ReadRepository) BulkInsertPromotions(promotions []*models.Promotion) error {
	if len(promotions) == 0 {
		return nil
	}

	// Start a transaction
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Prepare the bulk insert query
	valueStrings := make([]string, 0, len(promotions))
	valueArgs := make([]interface{}, 0, len(promotions)*3)
	for i, p := range promotions {
		valueStrings = append(valueStrings, fmt.Sprintf("($%d, $%d, $%d)", i*3+1, i*3+2, i*3+3))
		valueArgs = append(valueArgs, p.ID, p.Price, p.ExpirationDate)
	}

	stmt := fmt.Sprintf("INSERT INTO promotions_temp (id, price, expiration_date) VALUES %s",
		strings.Join(valueStrings, ","))

	// Execute the bulk insert
	_, err = tx.Exec(stmt, valueArgs...)
	if err != nil {
		return fmt.Errorf("failed to insert promotions: %w", err)
	}

	// Commit the transaction
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (r *ReadRepository) SwapTables() error {
	_, err := r.db.Exec(`
        ALTER TABLE promotions RENAME TO promotions_old;
        ALTER TABLE promotions_temp RENAME TO promotions;
        ALTER TABLE promotions_old RENAME TO promotions_temp;
        TRUNCATE TABLE promotions_temp;
    `)
	return err
}


func (r *ReadRepository) GetPromotion(id string) (*models.Promotion, error) {
	ctx := context.Background()

	// Try to get from cache first
	if r.cache != nil {
		cachedPromotion, err := r.cache.Get(ctx, id).Result()
		if err == nil {
			metrics.CacheHits.Inc()
			var promotion models.Promotion
			err = json.Unmarshal([]byte(cachedPromotion), &promotion)
			if err == nil {
				return &promotion, nil
			}
			logging.Logger.Error("Error unmarshalling cached promotion", zap.Error(err))
		} else if err != redis.Nil {
			logging.Logger.Error("Redis error", zap.Error(err))
		}
	}

	metrics.CacheMisses.Inc()

	// If not in cache or cache failed, get from database
	var promotion models.Promotion
	err := r.db.QueryRow("SELECT id, price, expiration_date FROM promotions WHERE id = $1", id).
		Scan(&promotion.ID, &promotion.Price, &promotion.ExpirationDate)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("promotion not found")
		}
		return nil, fmt.Errorf("database error: %w", err)
	}

	metrics.DatabaseOperations.WithLabelValues("read").Inc()

	// Store in cache for future requests
	if r.cache != nil {
		promotionJSON, _ := json.Marshal(promotion)
		err = r.cache.Set(ctx, id, promotionJSON, time.Hour).Err()
		if err != nil {
			logging.Logger.Error("Error setting promotion in cache", zap.Error(err))
		}
	}

	return &promotion, nil
}


