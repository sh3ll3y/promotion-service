package kafka

import (
	"encoding/json"
	"github.com/IBM/sarama"
)

type Producer struct {
	producer sarama.SyncProducer
	topic    string
}

func NewProducer(brokers []string, topic string) (*Producer, error) {
	config := sarama.NewConfig()
	config.Producer.Return.Successes = true
	producer, err := sarama.NewSyncProducer(brokers, config)
	if err != nil {
		return nil, err
	}
	return &Producer{producer: producer, topic: topic}, nil
}

func (p *Producer) PublishNewFileLoadedEvent() error {
	event := struct {
		Type string `json:"type"`
	}{
		Type: "NewFileLoaded",
	}

	eventJSON, err := json.Marshal(event)
	if err != nil {
		return err
	}

	msg := &sarama.ProducerMessage{
		Topic: p.topic,
		Value: sarama.StringEncoder(eventJSON),
	}

	_, _, err = p.producer.SendMessage(msg)
	return err
}