package kafka

import (
	"strings"

	"github.com/confluentinc/confluent-kafka-go/kafka"
)

type Kafka struct {
	producer *kafka.Producer
	consumer *kafka.Consumer
}

func NewKafka(brokers []string) (*Kafka, error) {
	producer, err := kafka.NewProducer(&kafka.ConfigMap{"bootstrap.servers": strings.Join(brokers, ",")})
	if err != nil {
		return nil, err
	}
	consumer, err := kafka.NewConsumer(&kafka.ConfigMap{"bootstrap.servers": strings.Join(brokers, ",")})
	if err != nil {
		return nil, err
	}
	return &Kafka{producer: producer, consumer: consumer}, nil
}

func (k *Kafka) Publish(topic string, message []byte) error {
	msg := &kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
		Value:          message,
	}
	err := k.producer.Produce(msg, nil)
	if err != nil {
		return err
	}
	return nil
}

func (k *Kafka) Subscribe(topic string, handler func(message []byte)) error {
	err := k.consumer.Subscribe(topic, nil)
	if err != nil {
		return err
	}
	defer k.consumer.Close()

	for msg := range k.consumer.Events() {
		switch e := msg.(type) {
		case *kafka.Message:
			handler(e.Value)
		case kafka.Error:
			return e
		}
	}
	return nil
}

func (k *Kafka) Close() error {
	return k.consumer.Close()
}
