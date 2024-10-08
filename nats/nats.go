package nats

import "github.com/nats-io/nats.go"

func ConnectNats(dns string) *nats.Conn {
	nc, _ := nats.Connect(dns)
	defer nc.Close()

	return nc
}
