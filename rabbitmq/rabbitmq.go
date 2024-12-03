package rabbitmq

import (
	"context"
	"log"
	"sync"

	"github.com/JIeeiroSst/utils/logger"
	"github.com/streadway/amqp"
)

var (
	once     sync.Once
	instance *RabbitMQConnect
)

type RabbitMQConnect struct {
	conn *amqp.Connection
}

func GetRabbitMQConnectInstance(dns string) *RabbitMQConnect {
	once.Do(func() {
		conn, err := amqp.Dial(dns)
		if err != nil {
			logger.ConfigZap(context.Background()).Sugar().Infof("Failed to connect to RabbitMQ: %v", err)
		}
		defer conn.Close()
		instance = &RabbitMQConnect{conn: conn}
	})
	return instance
}

func (r *RabbitMQConnect) InitQueue(ctx context.Context, queues []string) error {
	ch, err := r.conn.Channel()
	if err != nil {
		log.Fatal(err)
	}
	defer ch.Close()

	for _, queueName := range queues {
		_, err := ch.QueueDeclare(
			queueName, // name
			true,      // durable
			false,     // delete when unused
			false,     // exclusive
			false,     // no-wait
			nil,       // arguments
		)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *RabbitMQConnect) Publish(ctx context.Context, qName string, msg []byte) error {
	channel, err := r.conn.Channel()
	if err != nil {
		return err
	}
	err = channel.Publish(
		"",    // exchange
		qName, // key
		false, // mandatory
		false, // immediate
		amqp.Publishing{
			ContentType: "application/octet-stream",
			Body:        msg,
		},
	)
	if err != nil {
		return err
	}
	return nil
}

func (r *RabbitMQConnect) Consume(qName string) (msgs <-chan amqp.Delivery, err error) {
	channel, err := r.conn.Channel()
	if err != nil {
		return nil, err
	}
	msgs, err = channel.Consume(
		qName, // queue
		"",    // consumer
		true,  // auto-ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)

	return
}
