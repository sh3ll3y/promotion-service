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
	"time"
)

type ReadRepository struct {
	db    *sql.DB
	cache *redis.Client
}

func NewReadRepository(db *sql.DB, cache *redis.Client) *ReadRepository {
	return &ReadRepository{db: db, cache: cache}
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

func (r *ReadRepository) UpsertPromotion(p *models.Promotion) error {
	_, err := r.db.Exec(`
        INSERT INTO promotions (id, price, expiration_date)
        VALUES ($1, $2, $3)
        ON CONFLICT (id) DO UPDATE SET
            price = EXCLUDED.price,
            expiration_date = EXCLUDED.expiration_date
    `, p.ID, p.Price, p.ExpirationDate)

	metrics.DatabaseOperations.WithLabelValues("upsert").Inc()

	return err
}