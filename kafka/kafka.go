package kafka

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/IBM/sarama"
)

type KafkaClient struct {
	brokers  []string
	config   *sarama.Config
	producer sarama.SyncProducer
}

func NewKafkaClient(brokers []string) (*KafkaClient, error) {
	config := sarama.NewConfig()
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Retry.Max = 5
	config.Producer.Return.Successes = true
	config.Version = sarama.V2_8_0_0

	producer, err := sarama.NewSyncProducer(brokers, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create producer: %w", err)
	}

	return &KafkaClient{
		brokers:  brokers,
		config:   config,
		producer: producer,
	}, nil
}

func (k *KafkaClient) Close() error {
	return k.producer.Close()
}

func (k *KafkaClient) SendMessage(topic string, key, value []byte) (int32, int64, error) {
	msg := &sarama.ProducerMessage{
		Topic: topic,
		Key:   sarama.ByteEncoder(key),
		Value: sarama.ByteEncoder(value),
	}

	return k.producer.SendMessage(msg)
}

type Consumer struct {
	client   sarama.Client
	consumer sarama.Consumer
	topics   []string
	group    string
	handler  func(message *sarama.ConsumerMessage) error
}

func NewConsumer(brokers []string, topics []string, handler func(message *sarama.ConsumerMessage) error) (*Consumer, error) {
	config := sarama.NewConfig()
	config.Consumer.Return.Errors = true
	config.Version = sarama.V2_8_0_0

	client, err := sarama.NewClient(brokers, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	consumer, err := sarama.NewConsumerFromClient(client)
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to create consumer: %w", err)
	}

	return &Consumer{
		client:   client,
		consumer: consumer,
		topics:   topics,
		handler:  handler,
	}, nil
}

func (c *Consumer) Consume(ctx context.Context) error {
	partitionConsumers := make([]sarama.PartitionConsumer, 0)

	for _, topic := range c.topics {
		partitions, err := c.consumer.Partitions(topic)
		if err != nil {
			return fmt.Errorf("failed to get partitions for topic %s: %w", topic, err)
		}

		for _, partition := range partitions {
			pc, err := c.consumer.ConsumePartition(topic, partition, sarama.OffsetNewest)
			if err != nil {
				return fmt.Errorf("failed to start consumer for partition %d: %w", partition, err)
			}
			partitionConsumers = append(partitionConsumers, pc)
		}
	}

	var wg sync.WaitGroup
	for _, pc := range partitionConsumers {
		wg.Add(1)
		go func(pc sarama.PartitionConsumer) {
			defer wg.Done()
			defer pc.Close()

			for {
				select {
				case msg := <-pc.Messages():
					if err := c.handler(msg); err != nil {
						log.Printf("Error handling message: %v", err)
					}
				case err := <-pc.Errors():
					log.Printf("Error consuming message: %v", err)
				case <-ctx.Done():
					return
				}
			}
		}(pc)
	}

	wg.Wait()
	return nil
}

func (c *Consumer) Close() error {
	if err := c.consumer.Close(); err != nil {
		return err
	}
	return c.client.Close()
}

type Subscriber struct {
	client        sarama.Client
	consumerGroup sarama.ConsumerGroup
	topics        []string
	groupID       string
	handler       func(message *sarama.ConsumerMessage) error
}

type ConsumerGroupHandler struct {
	handler func(message *sarama.ConsumerMessage) error
}

func (h ConsumerGroupHandler) Setup(_ sarama.ConsumerGroupSession) error {
	return nil
}

func (h ConsumerGroupHandler) Cleanup(_ sarama.ConsumerGroupSession) error {
	return nil
}

func (h ConsumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for message := range claim.Messages() {
		if err := h.handler(message); err != nil {
			log.Printf("Error processing message: %v", err)
		}
		session.MarkMessage(message, "")
	}
	return nil
}

func NewSubscriber(brokers []string, topics []string, groupID string, handler func(message *sarama.ConsumerMessage) error) (*Subscriber, error) {
	config := sarama.NewConfig()
	config.Version = sarama.V2_8_0_0
	config.Consumer.Return.Errors = true
	config.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategyRoundRobin
	config.Consumer.Offsets.Initial = sarama.OffsetNewest

	client, err := sarama.NewClient(brokers, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	consumerGroup, err := sarama.NewConsumerGroupFromClient(groupID, client)
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to create consumer group: %w", err)
	}

	return &Subscriber{
		client:        client,
		consumerGroup: consumerGroup,
		topics:        topics,
		groupID:       groupID,
		handler:       handler,
	}, nil
}

func (s *Subscriber) Subscribe(ctx context.Context) error {
	handler := ConsumerGroupHandler{
		handler: s.handler,
	}

	go func() {
		for err := range s.consumerGroup.Errors() {
			log.Printf("Consumer group error: %v", err)
		}
	}()

	for {
		if err := s.consumerGroup.Consume(ctx, s.topics, handler); err != nil {
			return fmt.Errorf("error from consumer group: %w", err)
		}

		if ctx.Err() != nil {
			return nil
		}
	}
}

func (s *Subscriber) Close() error {
	if err := s.consumerGroup.Close(); err != nil {
		return err
	}
	return s.client.Close()
}
