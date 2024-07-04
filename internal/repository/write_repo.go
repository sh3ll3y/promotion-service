package repository

import (
	"database/sql"
	_ "github.com/lib/pq"
	"github.com/sh3ll3y/promotion-service/internal/metrics"
	"github.com/sh3ll3y/promotion-service/internal/models"
)

type WriteRepository struct {
	db *sql.DB
}

func NewWriteRepository(db *sql.DB) (*WriteRepository, error) {
	return &WriteRepository{db: db}, nil
}

func (r *WriteRepository) BeginTx() (*sql.Tx, error) {
	return r.db.Begin()
}

func (r *WriteRepository) CreatePromotionTx(tx *sql.Tx, p *models.Promotion) error {
	_, err := tx.Exec("INSERT INTO promotions (id, price, expiration_date) VALUES ($1, $2, $3)",
		p.ID, p.Price, p.ExpirationDate)
	if err != nil {
		return err
	}
	metrics.DatabaseOperations.WithLabelValues("create").Inc()
	return nil
}

// Keep the non-transactional method for compatibility
func (r *WriteRepository) CreatePromotion(p *models.Promotion) error {
	_, err := r.db.Exec("INSERT INTO promotions (id, price, expiration_date) VALUES ($1, $2, $3)",
		p.ID, p.Price, p.ExpirationDate)
	if err != nil {
		return err
	}
	metrics.DatabaseOperations.WithLabelValues("create").Inc()
	return nil
}

func (r *WriteRepository) BulkCreatePromotions(promotions []*models.Promotion) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare("INSERT INTO promotions (id, price, expiration_date) VALUES ($1, $2, $3)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, p := range promotions {
		_, err = stmt.Exec(p.ID, p.Price, p.ExpirationDate)
		if err != nil {
			return err
		}
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	metrics.DatabaseOperations.WithLabelValues("bulk_create").Add(float64(len(promotions)))
	return nil
}