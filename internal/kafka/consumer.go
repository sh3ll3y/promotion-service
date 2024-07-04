package kafka

import (
	"encoding/json"
	"github.com/IBM/sarama"
	"github.com/sh3ll3y/promotion-service/internal/logging"
	"github.com/sh3ll3y/promotion-service/internal/models"
	"github.com/sh3ll3y/promotion-service/internal/repository"
	"go.uber.org/zap"
)

func StartConsumer(brokers []string, topic string, readRepo *repository.ReadRepository) {
	config := sarama.NewConfig()
	consumer, err := sarama.NewConsumer(brokers, config)
	if err != nil {
		logging.Logger.Fatal("Error creating consumer", zap.Error(err))
	}
	defer consumer.Close()

	partitionConsumer, err := consumer.ConsumePartition(topic, 0, sarama.OffsetNewest)
	if err != nil {
		logging.Logger.Fatal("Error creating partition consumer", zap.Error(err))
	}
	defer partitionConsumer.Close()

	logging.Logger.Info("Kafka consumer started", zap.Strings("brokers", brokers), zap.String("topic", topic))

	for message := range partitionConsumer.Messages() {
		var promotion models.Promotion
		err := json.Unmarshal(message.Value, &promotion)
		if err != nil {
			logging.Logger.Error("Error unmarshalling message", zap.Error(err))
			continue
		}

		err = readRepo.UpsertPromotion(&promotion)
		if err != nil {
			logging.Logger.Error("Error upserting promotion in read database", zap.Error(err))
		}
	}
}