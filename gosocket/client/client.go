package client

import (
	"context"
	"sync"
	"time"

	"github.com/JIeeiroSst/utils/gosocket/codec"
	"github.com/JIeeiroSst/utils/gosocket/core"
	"github.com/gorilla/websocket"
)

type Config struct {
	URL string
	Codec core.Codec
	ReconnectInterval time.Duration
	MaxReconnectAttempts int
	PingInterval time.Duration
	PongWait time.Duration
	OnConnect func()
	OnDisconnect func()
	OnError func(err error)
	OnReconnecting func(attempt int)
}

func DefaultClientConfig() Config {
	return Config{
		Codec:                codec.NewJSONCodec(),
		ReconnectInterval:    5 * time.Second,
		MaxReconnectAttempts: 0,
		PingInterval:         30 * time.Second,
		PongWait:             60 * time.Second,
	}
}

type Client struct {
	config    Config
	conn      *websocket.Conn
	handlers  map[string][]core.Handler
	mu        sync.RWMutex
	sendCh    chan *core.Message
	ctx       context.Context
	cancel    context.CancelFunc
	connected bool
	connMu    sync.RWMutex
}

func New(cfg Config) *Client {
	if cfg.Codec == nil {
		cfg.Codec = codec.NewJSONCodec()
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Client{
		config:   cfg,
		handlers: make(map[string][]core.Handler),
		sendCh:   make(chan *core.Message, 256),
		ctx:      ctx,
		cancel:   cancel,
	}
}

func (c *Client) Connect() error {
	conn, _, err := websocket.DefaultDialer.Dial(c.config.URL, nil)
	if err != nil {
		return err
	}

	c.connMu.Lock()
	c.conn = conn
	c.connected = true
	c.connMu.Unlock()

	if c.config.OnConnect != nil {
		c.config.OnConnect()
	}

	go c.readPump()
	go c.writePump()

	return nil
}

func (c *Client) ConnectWithRetry() error {
	attempt := 0

	for {
		err := c.Connect()
		if err == nil {
			return nil
		}

		attempt++
		if c.config.MaxReconnectAttempts > 0 && attempt >= c.config.MaxReconnectAttempts {
			return err
		}

		if c.config.OnReconnecting != nil {
			c.config.OnReconnecting(attempt)
		}

		time.Sleep(c.config.ReconnectInterval)
	}
}

func (c *Client) On(event string, handler core.Handler) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, ok := c.handlers[event]; !ok {
		c.handlers[event] = make([]core.Handler, 0)
	}

	c.handlers[event] = append(c.handlers[event], handler)
}

func (c *Client) OnFunc(event string, fn func(ctx *core.HandlerContext) error) {
	c.On(event, core.HandlerFunc(fn))
}

func (c *Client) Emit(event string, data interface{}) error {
	msg := &core.Message{
		Event:     event,
		Data:      data,
		Timestamp: time.Now(),
	}

	select {
	case c.sendCh <- msg:
		return nil
	case <-c.ctx.Done():
		return c.ctx.Err()
	}
}

func (c *Client) Close() error {
	c.cancel()

	c.connMu.Lock()
	defer c.connMu.Unlock()

	if c.conn != nil {
		return c.conn.Close()
	}

	return nil
}

func (c *Client) IsConnected() bool {
	c.connMu.RLock()
	defer c.connMu.RUnlock()
	return c.connected
}

func (c *Client) readPump() {
	defer func() {
		c.connMu.Lock()
		c.connected = false
		c.connMu.Unlock()

		if c.config.OnDisconnect != nil {
			c.config.OnDisconnect()
		}

		if c.config.ReconnectInterval > 0 {
			go c.reconnect()
		}
	}()

	c.conn.SetReadDeadline(time.Now().Add(c.config.PongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(c.config.PongWait))
		return nil
	})

	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			if c.config.OnError != nil {
				c.config.OnError(err)
			}
			break
		}

		msg, err := c.config.Codec.Decode(data)
		if err != nil {
			if c.config.OnError != nil {
				c.config.OnError(err)
			}
			continue
		}

		go c.handleMessage(msg)
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(c.config.PingInterval)
	defer ticker.Stop()

	for {
		select {
		case msg := <-c.sendCh:
			data, err := c.config.Codec.Encode(msg)
			if err != nil {
				if c.config.OnError != nil {
					c.config.OnError(err)
				}
				continue
			}

			if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
				if c.config.OnError != nil {
					c.config.OnError(err)
				}
				return
			}

		case <-ticker.C:
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}

		case <-c.ctx.Done():
			return
		}
	}
}

func (c *Client) handleMessage(msg *core.Message) {
	c.mu.RLock()
	handlers, ok := c.handlers[msg.Event]
	c.mu.RUnlock()

	if !ok {
		return
	}

	ctx := core.NewHandlerContext(c.ctx, &dummyConnection{}, msg)

	for _, handler := range handlers {
		if err := handler.Handle(ctx); err != nil {
			if c.config.OnError != nil {
				c.config.OnError(err)
			}
		}
	}
}

func (c *Client) reconnect() {
	attempt := 0

	for {
		attempt++

		if c.config.MaxReconnectAttempts > 0 && attempt > c.config.MaxReconnectAttempts {
			return
		}

		if c.config.OnReconnecting != nil {
			c.config.OnReconnecting(attempt)
		}

		time.Sleep(c.config.ReconnectInterval)

		if err := c.Connect(); err == nil {
			return
		}
	}
}

type dummyConnection struct{}

func (d *dummyConnection) ID() string                         { return "client" }
func (d *dummyConnection) Send(msg *core.Message) error       { return nil }
func (d *dummyConnection) Close() error                       { return nil }
func (d *dummyConnection) Context() context.Context           { return context.Background() }
func (d *dummyConnection) Set(key string, value interface{})  {}
func (d *dummyConnection) Get(key string) (interface{}, bool) { return nil, false }
func (d *dummyConnection) RemoteAddr() string                 { return "" }
