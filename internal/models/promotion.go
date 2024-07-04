package models

import "time"

type Promotion struct {
	ID             string    `json:"id"`
	Price          float64   `json:"price"`
	ExpirationDate time.Time `json:"expiration_date"`
}