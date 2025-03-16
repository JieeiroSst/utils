package rabbitmq

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/streadway/amqp"
)

type Connection struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	uri     string
}

type ExchangeType string

const (
	DirectExchange ExchangeType = "direct"
	DefaultExchange ExchangeType = ""
	FanoutExchange ExchangeType = "fanout"
	TopicExchange ExchangeType = "topic"
	HeadersExchange ExchangeType = "headers"
)

type Config struct {
	Host     string
	Port     int
	Username string
	Password string
	Vhost    string
}

func NewConnection(config Config) (*Connection, error) {
	uri := fmt.Sprintf("amqp://%s:%s@%s:%d/%s",
		config.Username,
		config.Password,
		config.Host,
		config.Port,
		config.Vhost)

	conn, err := amqp.Dial(uri)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	channel, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	return &Connection{
		conn:    conn,
		channel: channel,
		uri:     uri,
	}, nil
}

func (c *Connection) Close() error {
	if c.channel != nil {
		if err := c.channel.Close(); err != nil {
			return fmt.Errorf("failed to close channel: %w", err)
		}
	}

	if c.conn != nil {
		if err := c.conn.Close(); err != nil {
			return fmt.Errorf("failed to close connection: %w", err)
		}
	}

	return nil
}

type Producer struct {
	conn         *Connection
	exchangeName string
	exchangeType ExchangeType
	routingKey   string
	mandatory    bool
	immediate    bool
	contentType  string
}

type ConsumerOptions struct {
	QueueName string
	AutoAck   bool
	Exclusive bool
	NoLocal   bool
	NoWait    bool
	Args      amqp.Table
}

type Consumer struct {
	conn         *Connection
	exchangeName string
	exchangeType ExchangeType
	queueName    string
	routingKey   string
	autoAck      bool
	exclusive    bool
	noLocal      bool
	noWait       bool
	args         amqp.Table
}

type SubscriberOptions struct {
	QueueName     string
	BindingKeys   []string
	ConsumerTag   string
	AutoAck       bool
	Exclusive     bool
	NoLocal       bool
	NoWait        bool
	Args          amqp.Table
	PrefetchCount int
	PrefetchSize  int
	Global        bool
}

type Subscriber struct {
	conn         *Connection
	exchangeName string
	exchangeType ExchangeType
	queueName    string
	bindingKeys  []string
	consumerTag  string
	autoAck      bool
	exclusive    bool
	noLocal      bool
	noWait       bool
	args         amqp.Table
}

func NewProducer(conn *Connection, exchangeName string, exchangeType ExchangeType) (*Producer, error) {
	if exchangeType != DefaultExchange {
		if err := conn.channel.ExchangeDeclare(
			string(exchangeName),
			string(exchangeType),
			true,  // durable
			false, // auto-delete
			false, // internal
			false, // no-wait
			nil,   // arguments
		); err != nil {
			return nil, fmt.Errorf("failed to declare exchange: %w", err)
		}
	}

	return &Producer{
		conn:         conn,
		exchangeName: exchangeName,
		exchangeType: exchangeType,
		routingKey:   "",
		mandatory:    false,
		immediate:    false,
		contentType:  "text/plain",
	}, nil
}

func (p *Producer) SetRoutingKey(routingKey string) *Producer {
	p.routingKey = routingKey
	return p
}

func (p *Producer) SetMandatory(mandatory bool) *Producer {
	p.mandatory = mandatory
	return p
}

func (p *Producer) SetImmediate(immediate bool) *Producer {
	p.immediate = immediate
	return p
}

func (p *Producer) SetContentType(contentType string) *Producer {
	p.contentType = contentType
	return p
}

func (p *Producer) Publish(ctx context.Context, body []byte, headers amqp.Table) error {
	msg := amqp.Publishing{
		DeliveryMode: amqp.Persistent,
		Timestamp:    time.Now(),
		ContentType:  p.contentType,
		Body:         body,
		Headers:      headers,
	}

	if err := p.conn.channel.Publish(
		p.exchangeName,
		p.routingKey,
		p.mandatory,
		p.immediate,
		msg,
	); err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	return nil
}

func NewConsumer(conn *Connection, exchangeName string, exchangeType ExchangeType, options ConsumerOptions) (*Consumer, error) {
	if exchangeType != DefaultExchange {
		if err := conn.channel.ExchangeDeclare(
			string(exchangeName),
			string(exchangeType),
			true,  // durable
			false, // auto-delete
			false, // internal
			false, // no-wait
			nil,   // arguments
		); err != nil {
			return nil, fmt.Errorf("failed to declare exchange: %w", err)
		}
	}

	queue, err := conn.channel.QueueDeclare(
		options.QueueName,
		true,              // durable
		false,             // auto-delete
		options.Exclusive, // exclusive
		options.NoWait,    // no-wait
		options.Args,      // arguments
	)
	if err != nil {
		return nil, fmt.Errorf("failed to declare queue: %w", err)
	}

	return &Consumer{
		conn:         conn,
		exchangeName: exchangeName,
		exchangeType: exchangeType,
		queueName:    queue.Name,
		routingKey:   "",
		autoAck:      options.AutoAck,
		exclusive:    options.Exclusive,
		noLocal:      options.NoLocal,
		noWait:       options.NoWait,
		args:         options.Args,
	}, nil
}

func (c *Consumer) SetRoutingKey(routingKey string) *Consumer {
	c.routingKey = routingKey
	return c
}

func (c *Consumer) BindQueue() error {
	if c.exchangeType == DefaultExchange {
		return nil
	}

	if err := c.conn.channel.QueueBind(
		c.queueName,
		c.routingKey,
		c.exchangeName,
		false, // no-wait
		nil,   // arguments
	); err != nil {
		return fmt.Errorf("failed to bind queue: %w", err)
	}

	return nil
}

func (c *Consumer) Consume(ctx context.Context) (<-chan amqp.Delivery, error) {
	deliveries, err := c.conn.channel.Consume(
		c.queueName,
		"",          // consumer tag
		c.autoAck,   // auto-ack
		c.exclusive, // exclusive
		c.noLocal,   // no-local
		c.noWait,    // no-wait
		c.args,      // arguments
	)
	if err != nil {
		return nil, fmt.Errorf("failed to start consuming: %w", err)
	}

	return deliveries, nil
}

func NewSubscriber(conn *Connection, exchangeName string, exchangeType ExchangeType, options SubscriberOptions) (*Subscriber, error) {
	if exchangeType != DefaultExchange {
		if err := conn.channel.ExchangeDeclare(
			string(exchangeName),
			string(exchangeType),
			true,  // durable
			false, // auto-delete
			false, // internal
			false, // no-wait
			nil,   // arguments
		); err != nil {
			return nil, fmt.Errorf("failed to declare exchange: %w", err)
		}
	}

	if options.PrefetchCount > 0 || options.PrefetchSize > 0 {
		if err := conn.channel.Qos(
			options.PrefetchCount,
			options.PrefetchSize,
			options.Global,
		); err != nil {
			return nil, fmt.Errorf("failed to set QoS: %w", err)
		}
	}

	queue, err := conn.channel.QueueDeclare(
		options.QueueName,
		true,              // durable
		false,             // auto-delete
		options.Exclusive, // exclusive
		options.NoWait,    // no-wait
		options.Args,      // arguments
	)
	if err != nil {
		return nil, fmt.Errorf("failed to declare queue: %w", err)
	}

	return &Subscriber{
		conn:         conn,
		exchangeName: exchangeName,
		exchangeType: exchangeType,
		queueName:    queue.Name,
		bindingKeys:  options.BindingKeys,
		consumerTag:  options.ConsumerTag,
		autoAck:      options.AutoAck,
		exclusive:    options.Exclusive,
		noLocal:      options.NoLocal,
		noWait:       options.NoWait,
		args:         options.Args,
	}, nil
}

func (s *Subscriber) BindQueue() error {
	if s.exchangeType == DefaultExchange {
		return nil
	}

	switch s.exchangeType {
	case DirectExchange, TopicExchange:
		for _, key := range s.bindingKeys {
			if err := s.conn.channel.QueueBind(
				s.queueName,
				key,
				s.exchangeName,
				false, // no-wait
				nil,   // arguments
			); err != nil {
				return fmt.Errorf("failed to bind queue with key %s: %w", key, err)
			}
		}
	case FanoutExchange:
		if err := s.conn.channel.QueueBind(
			s.queueName,
			"", 
			s.exchangeName,
			false, 
			nil,   
		); err != nil {
			return fmt.Errorf("failed to bind queue to fanout exchange: %w", err)
		}
	case HeadersExchange:
		if err := s.conn.channel.QueueBind(
			s.queueName,
			"", 
			s.exchangeName,
			false,  // no-wait
			s.args, // arguments
		); err != nil {
			return fmt.Errorf("failed to bind queue to headers exchange: %w", err)
		}
	}

	return nil
}

func (s *Subscriber) Subscribe(ctx context.Context) (<-chan amqp.Delivery, error) {
	deliveries, err := s.conn.channel.Consume(
		s.queueName,
		s.consumerTag, // consumer tag
		s.autoAck,     // auto-ack
		s.exclusive,   // exclusive
		s.noLocal,     // no-local
		s.noWait,      // no-wait
		s.args,        // arguments
	)
	if err != nil {
		return nil, fmt.Errorf("failed to start subscribing: %w", err)
	}

	return deliveries, nil
}

func (p *Producer) PublishWithHeaders(ctx context.Context, body []byte, headers amqp.Table) error {
	msg := amqp.Publishing{
		DeliveryMode: amqp.Persistent,
		Timestamp:    time.Now(),
		ContentType:  p.contentType,
		Body:         body,
		Headers:      headers,
	}

	if err := p.conn.channel.Publish(
		p.exchangeName,
		p.routingKey,
		p.mandatory,
		p.immediate,
		msg,
	); err != nil {
		return fmt.Errorf("failed to publish message with headers: %w", err)
	}

	return nil
}

func (s *Subscriber) Cancel() error {
	if s.consumerTag != "" {
		if err := s.conn.channel.Cancel(s.consumerTag, false); err != nil {
			return fmt.Errorf("failed to cancel subscription: %w", err)
		}
	}
	return nil
}

func (s *Subscriber) GetQueueName() string {
	return s.queueName
}

func (c *Consumer) GetQueueName() string {
	return c.queueName
}

func ProcessMessages(ctx context.Context, deliveries <-chan amqp.Delivery, handler func(amqp.Delivery) error) {
	for {
		select {
		case <-ctx.Done():
			log.Println("Context cancelled, stopping message processing")
			return
		case delivery, ok := <-deliveries:
			if !ok {
				log.Println("Delivery channel closed, stopping message processing")
				return
			}

			if err := handler(delivery); err != nil {
				log.Printf("Error handling message: %v", err)

				if err := delivery.Nack(false, true); err != nil {
					log.Printf("Error nacking message: %v", err)
				}
			} else {
				if err := delivery.Ack(false); err != nil {
					log.Printf("Error acking message: %v", err)
				}
			}
		}
	}
}

func (c *Connection) ExchangeDeclare(name string, exchangeType ExchangeType, durable, autoDelete, internal, noWait bool, args amqp.Table) error {
	return c.channel.ExchangeDeclare(
		name,
		string(exchangeType),
		durable,
		autoDelete,
		internal,
		noWait,
		args,
	)
}

func (c *Connection) QueueDeclare(name string, durable, autoDelete, exclusive, noWait bool, args amqp.Table) (amqp.Queue, error) {
	return c.channel.QueueDeclare(
		name,
		durable,
		autoDelete,
		exclusive,
		noWait,
		args,
	)
}

func (c *Connection) QueueBind(queueName, routingKey, exchangeName string, noWait bool, args amqp.Table) error {
	return c.channel.QueueBind(
		queueName,
		routingKey,
		exchangeName,
		noWait,
		args,
	)
}

type ReconnectHandler struct {
	config        Config
	conn          *Connection
	maxRetries    int
	retryInterval time.Duration
}

func NewReconnectHandler(config Config, maxRetries int, retryInterval time.Duration) *ReconnectHandler {
	return &ReconnectHandler{
		config:        config,
		maxRetries:    maxRetries,
		retryInterval: retryInterval,
	}
}

func (r *ReconnectHandler) Connect() (*Connection, error) {
	var err error
	var conn *Connection

	for i := 0; i < r.maxRetries; i++ {
		conn, err = NewConnection(r.config)
		if err == nil {
			r.conn = conn
			return conn, nil
		}

		log.Printf("Failed to connect to RabbitMQ (attempt %d/%d): %v", i+1, r.maxRetries, err)
		time.Sleep(r.retryInterval)
	}

	return nil, fmt.Errorf("failed to connect to RabbitMQ after %d attempts: %w", r.maxRetries, err)
}

func (r *ReconnectHandler) Reconnect() (*Connection, error) {
	if r.conn != nil {
		_ = r.conn.Close() 
	}

	return r.Connect()
}

func (r *ReconnectHandler) MonitorConnection(ctx context.Context, onReconnect func(*Connection) error) {
	if r.conn == nil {
		log.Println("Cannot monitor: no connection established")
		return
	}

	closeChan := r.conn.conn.NotifyClose(make(chan *amqp.Error))

	for {
		select {
		case <-ctx.Done():
			log.Println("Context cancelled, stopping connection monitor")
			return
		case err, ok := <-closeChan:
			if !ok {
				log.Println("Connection closed channel closed")
				return
			}

			log.Printf("Connection closed: %v", err)

			var newConn *Connection
			var reconnectErr error

			for i := 0; i < r.maxRetries; i++ {
				log.Printf("Attempting to reconnect (attempt %d/%d)", i+1, r.maxRetries)
				newConn, reconnectErr = r.Reconnect()
				if reconnectErr == nil {
					break
				}
				time.Sleep(r.retryInterval)
			}

			if reconnectErr != nil {
				log.Printf("Failed to reconnect after %d attempts: %v", r.maxRetries, reconnectErr)
				return
			}

			log.Println("Successfully reconnected to RabbitMQ")

			if onReconnect != nil {
				if err := onReconnect(newConn); err != nil {
					log.Printf("Error in onReconnect callback: %v", err)
				}
			}

			// Update connection and notification channel
			r.conn = newConn
			closeChan = newConn.conn.NotifyClose(make(chan *amqp.Error))
		}
	}
}

type MessageHandler func(delivery amqp.Delivery) error

type Worker struct {
	subscriber  *Subscriber
	handler     MessageHandler
	concurrency int
	deliveries  <-chan amqp.Delivery
	autoAck     bool
}

func NewWorker(subscriber *Subscriber, handler MessageHandler, concurrency int, autoAck bool) *Worker {
	return &Worker{
		subscriber:  subscriber,
		handler:     handler,
		concurrency: concurrency,
		autoAck:     autoAck,
	}
}

func (w *Worker) Start(ctx context.Context) error {
	var err error
	w.deliveries, err = w.subscriber.Subscribe(ctx)
	if err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}

	for i := 0; i < w.concurrency; i++ {
		go func(workerID int) {
			log.Printf("Worker %d started", workerID)
			for {
				select {
				case <-ctx.Done():
					log.Printf("Worker %d stopped: context cancelled", workerID)
					return
				case delivery, ok := <-w.deliveries:
					if !ok {
						log.Printf("Worker %d stopped: channel closed", workerID)
						return
					}

					if err := w.handler(delivery); err != nil {
						log.Printf("Worker %d error: %v", workerID, err)
						if !w.autoAck {
							if err := delivery.Nack(false, true); err != nil {
								log.Printf("Worker %d error nacking message: %v", workerID, err)
							}
						}
					} else {
						if !w.autoAck {
							if err := delivery.Ack(false); err != nil {
								log.Printf("Worker %d error acking message: %v", workerID, err)
							}
						}
					}
				}
			}
		}(i)
	}

	return nil
}

func (w *Worker) Stop() error {
	return w.subscriber.Cancel()
}

type HeadersBindingType string

const (
	All HeadersBindingType = "all"
	Any HeadersBindingType = "any"
)

func HeadersBinding(bindingType HeadersBindingType, headers map[string]interface{}) amqp.Table {
	table := amqp.Table{}

	for k, v := range headers {
		table[k] = v
	}

	table["x-match"] = string(bindingType)

	return table
}

type BatchPublisher struct {
	producer *Producer
	batch    []amqp.Publishing
	size     int
}

func NewBatchPublisher(producer *Producer, size int) *BatchPublisher {
	return &BatchPublisher{
		producer: producer,
		batch:    make([]amqp.Publishing, 0, size),
		size:     size,
	}
}

func (b *BatchPublisher) Add(body []byte, headers amqp.Table) {
	msg := amqp.Publishing{
		DeliveryMode: amqp.Persistent,
		Timestamp:    time.Now(),
		ContentType:  b.producer.contentType,
		Body:         body,
		Headers:      headers,
	}

	b.batch = append(b.batch, msg)
}

func (b *BatchPublisher) Publish(ctx context.Context) (int, error) {
	if len(b.batch) == 0 {
		return 0, nil
	}

	count := len(b.batch)

	for _, msg := range b.batch {
		if err := b.producer.conn.channel.Publish(
			b.producer.exchangeName,
			b.producer.routingKey,
			b.producer.mandatory,
			b.producer.immediate,
			msg,
		); err != nil {
			return 0, fmt.Errorf("failed to publish batch message: %w", err)
		}
	}

	b.batch = b.batch[:0] // Clear the batch

	return count, nil
}

func (b *BatchPublisher) IsFull() bool {
	return len(b.batch) >= b.size
}

func (b *BatchPublisher) Size() int {
	return len(b.batch)
}

func (b *BatchPublisher) Clear() {
	b.batch = b.batch[:0]
}
