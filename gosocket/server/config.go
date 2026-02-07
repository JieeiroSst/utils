package server

import (
	"time"

	"github.com/JIeeiroSst/utils/gosocket/codec"
	"github.com/JIeeiroSst/utils/gosocket/core"
)

type Config struct {
	Addr string
	Path string
	ReadBufferSize int
	WriteBufferSize int
	HandshakeTimeout time.Duration
	WriteTimeout time.Duration
	ReadTimeout time.Duration
	PingInterval time.Duration
	PongWait time.Duration
	MaxMessageSize int64
	Codec core.Codec
	CheckOrigin func(r interface{}) bool
	OnConnect func(conn core.Connection)
	OnDisconnect func(conn core.Connection)
	OnError func(conn core.Connection, err error)
}

func DefaultConfig() Config {
	return Config{
		Addr:             ":8080",
		Path:             "/ws",
		ReadBufferSize:   4096,
		WriteBufferSize:  4096,
		HandshakeTimeout: 10 * time.Second,
		WriteTimeout:     10 * time.Second,
		ReadTimeout:      60 * time.Second,
		PingInterval:     30 * time.Second,
		PongWait:         60 * time.Second,
		MaxMessageSize:   512 * 1024, 
		Codec:            codec.NewJSONCodec(),
		CheckOrigin: func(r interface{}) bool {
			return true 
		},
	}
}

func (c Config) Merge() Config {
	def := DefaultConfig()

	if c.Addr == "" {
		c.Addr = def.Addr
	}
	if c.Path == "" {
		c.Path = def.Path
	}
	if c.ReadBufferSize == 0 {
		c.ReadBufferSize = def.ReadBufferSize
	}
	if c.WriteBufferSize == 0 {
		c.WriteBufferSize = def.WriteBufferSize
	}
	if c.HandshakeTimeout == 0 {
		c.HandshakeTimeout = def.HandshakeTimeout
	}
	if c.WriteTimeout == 0 {
		c.WriteTimeout = def.WriteTimeout
	}
	if c.ReadTimeout == 0 {
		c.ReadTimeout = def.ReadTimeout
	}
	if c.PingInterval == 0 {
		c.PingInterval = def.PingInterval
	}
	if c.PongWait == 0 {
		c.PongWait = def.PongWait
	}
	if c.MaxMessageSize == 0 {
		c.MaxMessageSize = def.MaxMessageSize
	}
	if c.Codec == nil {
		c.Codec = def.Codec
	}
	if c.CheckOrigin == nil {
		c.CheckOrigin = def.CheckOrigin
	}

	return c
}
