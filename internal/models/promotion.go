package models

import (
	"github.com/go-playground/validator/v10"
	"time"
)

type Promotion struct {
	ID             string    `json:"id" validate:"required,uuid4"`
	Price          float64   `json:"price" validate:"required,gt=0"`
	ExpirationDate time.Time `json:"expiration_date" validate:"required"`
}

var validate = validator.New()

func (p *Promotion) Validate() error {
	return validate.Struct(p)
}