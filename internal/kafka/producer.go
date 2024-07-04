package kafka

import (
	"encoding/json"
	"github.com/IBM/sarama"
	"github.com/sh3ll3y/promotion-service/internal/models"
	"github.com/sh3ll3y/promotion-service/internal/retry"
	"time"
)

type Producer struct {
	producer sarama.SyncProducer
	topic    string
}

func NewProducer(brokers []string) (*Producer, error) {
	config := sarama.NewConfig()
	config.Producer.Return.Successes = true
	producer, err := sarama.NewSyncProducer(brokers, config)
	if err != nil {
		return nil, err
	}
	return &Producer{producer: producer, topic: "promotions"}, nil
}

func (p *Producer) PublishPromotion(promotion *models.Promotion) error {
	value, err := json.Marshal(promotion)
	if err != nil {
		return err
	}

	msg := &sarama.ProducerMessage{
		Topic: p.topic,
		Value: sarama.StringEncoder(value),
	}

	return retry.Do(3, time.Second, func() error {
		_, _, err := p.producer.SendMessage(msg)
		return err
	})
}