package rabbitmq

import (
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
			logger.ConfigZap().Errorf("Failed to connect to RabbitMQ: %v", err)
		}
		defer conn.Close()
		instance = &RabbitMQConnect{conn: conn}
	})
	return instance
}
