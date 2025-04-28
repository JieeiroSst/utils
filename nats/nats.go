package nats

import (
	"fmt"

	"github.com/nats-io/nats.go"
)

type NatsConnection interface {
	PublishMessage(subject string, data []byte) error
	SubscribeToSubject(subject string, handler func([]byte)) (*nats.Subscription, error)
	CloseSubscription(sub *nats.Subscription) error
}

type natsConnection struct {
	nc *nats.Conn
}

func InitializeNatsConnection(dns string) NatsConnection {
	nc, _ := nats.Connect(dns)
	defer nc.Close()

	return &natsConnection{nc: nc}
}

func (n *natsConnection) PublishMessage(subject string, data []byte) error {
	if n.nc == nil || !n.nc.IsConnected() {
		return fmt.Errorf("kết nối NATS không khả dụng")
	}

	err := n.nc.Publish(subject, data)
	if err != nil {
		return fmt.Errorf("không thể đăng tải tin nhắn: %v", err)
	}

	err = n.nc.Flush()
	if err != nil {
		return fmt.Errorf("không thể flush tin nhắn: %v", err)
	}

	return nil
}

func (n *natsConnection) SubscribeToSubject(subject string, handler func([]byte)) (*nats.Subscription, error) {
	if n.nc == nil || !n.nc.IsConnected() {
		return nil, fmt.Errorf("kết nối NATS không khả dụng")
	}

	sub, err := n.nc.Subscribe(subject, func(msg *nats.Msg) {
		handler(msg.Data)
	})

	if err != nil {
		return nil, fmt.Errorf("không thể đăng ký subject %s: %v", subject, err)
	}

	return sub, nil
}

func (n *natsConnection) CloseSubscription(sub *nats.Subscription) error {
	if sub == nil {
		return fmt.Errorf("subscription không hợp lệ")
	}

	return sub.Unsubscribe()
}
