package rabbitmq

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/streadway/amqp"
)

var (
	ErrNotConnected = errors.New("not connected to RabbitMQ")
	ErrAlreadyConnected = errors.New("already connected to RabbitMQ")
	ErrConnectionClosed = errors.New("connection closed")
)

type RabbitMQ struct {
	conn         *amqp.Connection
	channel      *amqp.Channel
	exchangeName string
	queueName    string
	routingKey   string
	uri          string
	connected    bool
	closed       chan struct{}
}

func NewRabbitMQ(uri, exchangeName, queueName, routingKey string) *RabbitMQ {
	return &RabbitMQ{
		uri:          uri,
		exchangeName: exchangeName,
		queueName:    queueName,
		routingKey:   routingKey,
		connected:    false,
		closed:       make(chan struct{}),
	}
}

func (r *RabbitMQ) Connect() error {
	if r.connected {
		return ErrAlreadyConnected
	}

	var err error
	r.conn, err = amqp.Dial(r.uri)
	if err != nil {
		return err
	}

	r.channel, err = r.conn.Channel()
	if err != nil {
		r.conn.Close()
		return err
	}

	err = r.channel.ExchangeDeclare(
		r.exchangeName, // name
		"topic",        // type
		true,           // durable
		false,          // auto-deleted
		false,          // internal
		false,          // no-wait
		nil,            // arguments
	)
	if err != nil {
		r.channel.Close()
		r.conn.Close()
		return err
	}

	_, err = r.channel.QueueDeclare(
		r.queueName, // name
		true,        // durable
		false,       // delete when unused
		false,       // exclusive
		false,       // no-wait
		nil,         // arguments
	)
	if err != nil {
		r.channel.Close()
		r.conn.Close()
		return err
	}

	err = r.channel.QueueBind(
		r.queueName,    // queue name
		r.routingKey,   // routing key
		r.exchangeName, // exchange
		false,
		nil,
	)
	if err != nil {
		r.channel.Close()
		r.conn.Close()
		return err
	}

	r.connected = true
	go r.handleReconnect()
	return nil
}

func (r *RabbitMQ) Close() error {
	if !r.connected {
		return ErrNotConnected
	}

	close(r.closed)
	err := r.channel.Close()
	if err != nil {
		return err
	}

	err = r.conn.Close()
	if err != nil {
		return err
	}

	r.connected = false
	return nil
}

func (r *RabbitMQ) handleReconnect() {
	connCloseChan := r.conn.NotifyClose(make(chan *amqp.Error))
	
	select {
	case <-connCloseChan:
		log.Println("Connection closed, attempting to reconnect...")
		for {
			if !r.connected {
				return
			}
			
			time.Sleep(5 * time.Second)
			
			err := r.Connect()
			if err == nil {
				log.Println("Reconnected to RabbitMQ")
				return
			}
			
			log.Printf("Failed to reconnect: %v", err)
		}
	case <-r.closed:
		return
	}
}

type Producer struct {
	*RabbitMQ
}

func NewProducer(uri, exchangeName, routingKey string) *Producer {
	return &Producer{
		RabbitMQ: NewRabbitMQ(uri, exchangeName, "", routingKey),
	}
}

func (p *Producer) Publish(contentType string, body []byte) error {
	if !p.connected {
		return ErrNotConnected
	}

	return p.channel.Publish(
		p.exchangeName, // exchange
		p.routingKey,   // routing key
		false,          // mandatory
		false,          // immediate
		amqp.Publishing{
			ContentType:  contentType,
			Body:         body,
			DeliveryMode: amqp.Persistent,
			Timestamp:    time.Now(),
		},
	)
}

type Consumer struct {
	*RabbitMQ
	deliveries <-chan amqp.Delivery
	handler    func([]byte) error
}

func NewConsumer(uri, exchangeName, queueName, routingKey string) *Consumer {
	return &Consumer{
		RabbitMQ: NewRabbitMQ(uri, exchangeName, queueName, routingKey),
	}
}

func (c *Consumer) Consume(handler func([]byte) error) error {
	if !c.connected {
		return ErrNotConnected
	}

	c.handler = handler

	var err error
	c.deliveries, err = c.channel.Consume(
		c.queueName, // queue
		"",          // consumer
		false,       // auto-ack
		false,       // exclusive
		false,       // no-local
		false,       // no-wait
		nil,         // args
	)
	if err != nil {
		return err
	}

	go c.handleDeliveries()
	return nil
}

func (c *Consumer) handleDeliveries() {
	for {
		select {
		case delivery, ok := <-c.deliveries:
			if !ok {
				log.Println("Delivery channel closed")
				return
			}

			err := c.handler(delivery.Body)
			if err != nil {
				log.Printf("Error handling message: %v", err)
				delivery.Nack(false, true)
			} else {
				delivery.Ack(false)
			}
		case <-c.closed:
			return
		}
	}
}

type Subscriber struct {
	*RabbitMQ
	patterns []string
	handlers map[string]func([]byte) error
}

func NewSubscriber(uri, exchangeName, queueName string) *Subscriber {
	return &Subscriber{
		RabbitMQ:  NewRabbitMQ(uri, exchangeName, queueName, ""),
		patterns:  make([]string, 0),
		handlers:  make(map[string]func([]byte) error),
	}
}

func (s *Subscriber) Subscribe(pattern string, handler func([]byte) error) error {
	if !s.connected {
		return ErrNotConnected
	}

	s.patterns = append(s.patterns, pattern)
	s.handlers[pattern] = handler

	err := s.channel.QueueBind(
		s.queueName,    // queue name
		pattern,        // routing key
		s.exchangeName, // exchange
		false,
		nil,
	)
	if err != nil {
		return err
	}

	return nil
}

func (s *Subscriber) Start(ctx context.Context) error {
	if !s.connected {
		return ErrNotConnected
	}

	deliveries, err := s.channel.Consume(
		s.queueName, // queue
		"",          // consumer
		false,       // auto-ack
		false,       // exclusive
		false,       // no-local
		false,       // no-wait
		nil,         // args
	)
	if err != nil {
		return err
	}

	go func() {
		for {
			select {
			case delivery, ok := <-deliveries:
				if !ok {
					log.Println("Delivery channel closed")
					return
				}

				routingKey := delivery.RoutingKey
				handler, exists := s.handlers[routingKey]
				
				if !exists {
					for pattern, h := range s.handlers {
						if matchTopic(pattern, routingKey) {
							handler = h
							exists = true
							break
						}
					}
				}

				if exists {
					err := handler(delivery.Body)
					if err != nil {
						log.Printf("Error handling message: %v", err)
						delivery.Nack(false, true)
					} else {
						delivery.Ack(false)
					}
				} else {
					delivery.Ack(false)
				}
			case <-ctx.Done():
				return
			case <-s.closed:
				return
			}
		}
	}()

	return nil
}

func matchTopic(pattern, routingKey string) bool {
	return pattern == routingKey || pattern == "#"
}

