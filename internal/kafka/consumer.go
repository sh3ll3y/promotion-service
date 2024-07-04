package kafka

import (
	"encoding/json"
	"github.com/IBM/sarama"
	"github.com/sh3ll3y/promotion-service/internal/logging"
	"github.com/sh3ll3y/promotion-service/internal/service"
	"go.uber.org/zap"
)

type Consumer struct {
	consumer sarama.Consumer
	topic    string
	service  *service.PromotionService
}

func NewConsumer(brokers []string, topic string, service *service.PromotionService) (*Consumer, error) {
	config := sarama.NewConfig()
	consumer, err := sarama.NewConsumer(brokers, config)
	if err != nil {
		return nil, err
	}
	return &Consumer{consumer: consumer, topic: topic, service: service}, nil
}

func (c *Consumer) Start() error {
	partitionConsumer, err := c.consumer.ConsumePartition(c.topic, 0, sarama.OffsetNewest)
	if err != nil {
		return err
	}

	for message := range partitionConsumer.Messages() {
		var event struct {
			Type string `json:"type"`
		}
		err := json.Unmarshal(message.Value, &event)
		if err != nil {
			logging.Logger.Error("Failed to unmarshal event", zap.Error(err))
			continue
		}

		if event.Type == "NewFileLoaded" {
			err := c.service.UpdateReadDB()
			if err != nil {
				logging.Logger.Error("Failed to update read DB", zap.Error(err))
			}
		}
	}

	return nil
}