package repository

import (
	"database/sql"
	"fmt"
	"github.com/sh3ll3y/promotion-service/internal/models"
)

type WriteRepository struct {
	db *sql.DB
}

func NewWriteRepository(db *sql.DB) *WriteRepository {
	return &WriteRepository{db: db}
}

func (r *WriteRepository) ClearAllPromotions() error {
	_, err := r.db.Exec("DELETE FROM promotions")
	return err
}

func (r *WriteRepository) CreatePromotion(p *models.Promotion) error {
	_, err := r.db.Exec(
		"INSERT INTO promotions (id, price, expiration_date) VALUES ($1, $2, $3)",
		p.ID, p.Price, p.ExpirationDate,
	)
	if err != nil {
		return fmt.Errorf("failed to insert promotion: %w", err)
	}
	return nil
}

func (r *WriteRepository) GetTotalPromotionsCount() (int, error) {
	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM promotions").Scan(&count)
	return count, err
}

func (r *WriteRepository) GetPromotionsBatch(offset, limit int) ([]*models.Promotion, error) {
	rows, err := r.db.Query("SELECT id, price, expiration_date FROM promotions LIMIT $1 OFFSET $2", limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var promotions []*models.Promotion
	for rows.Next() {
		p := &models.Promotion{}
		err := rows.Scan(&p.ID, &p.Price, &p.ExpirationDate)
		if err != nil {
			return nil, err
		}
		promotions = append(promotions, p)
	}

	return promotions, rows.Err()
}