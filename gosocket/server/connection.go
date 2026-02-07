package server

import (
	"context"
	"sync"
	"time"

	"github.com/JIeeiroSst/utils/gosocket/core"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type Conn struct {
	id        string
	ws        *websocket.Conn
	ctx       context.Context
	cancel    context.CancelFunc
	server    *Server
	sendCh    chan *core.Message
	closeCh   chan struct{}
	closeOnce sync.Once
	data      map[string]interface{}
	dataMu    sync.RWMutex
}

func NewConn(ws *websocket.Conn, server *Server) *Conn {
	ctx, cancel := context.WithCancel(context.Background())

	conn := &Conn{
		id:      uuid.New().String(),
		ws:      ws,
		ctx:     ctx,
		cancel:  cancel,
		server:  server,
		sendCh:  make(chan *core.Message, 256),
		closeCh: make(chan struct{}),
		data:    make(map[string]interface{}),
	}

	return conn
}

func (c *Conn) ID() string {
	return c.id
}

func (c *Conn) Send(msg *core.Message) error {
	select {
	case c.sendCh <- msg:
		return nil
	case <-c.closeCh:
		return ErrConnectionClosed
	case <-c.ctx.Done():
		return c.ctx.Err()
	}
}

func (c *Conn) Close() error {
	var err error
	c.closeOnce.Do(func() {
		close(c.closeCh)
		c.cancel()
		err = c.ws.Close()
	})
	return err
}

func (c *Conn) Context() context.Context {
	return c.ctx
}

func (c *Conn) Set(key string, value interface{}) {
	c.dataMu.Lock()
	defer c.dataMu.Unlock()
	c.data[key] = value
}

func (c *Conn) Get(key string) (interface{}, bool) {
	c.dataMu.RLock()
	defer c.dataMu.RUnlock()
	val, ok := c.data[key]
	return val, ok
}

func (c *Conn) RemoteAddr() string {
	return c.ws.RemoteAddr().String()
}

func (c *Conn) readPump() {
	defer func() {
		c.server.unregister <- c
		c.Close()
	}()

	c.ws.SetReadDeadline(time.Now().Add(c.server.config.PongWait))
	c.ws.SetPongHandler(func(string) error {
		c.ws.SetReadDeadline(time.Now().Add(c.server.config.PongWait))
		return nil
	})

	for {
		_, data, err := c.ws.ReadMessage()
		if err != nil {
			if c.server.config.OnError != nil {
				c.server.config.OnError(c, err)
			}
			break
		}

		msg, err := c.server.config.Codec.Decode(data)
		if err != nil {
			if c.server.config.OnError != nil {
				c.server.config.OnError(c, err)
			}
			continue
		}

		if msg.Timestamp.IsZero() {
			msg.Timestamp = time.Now()
		}

		go c.server.handleMessage(c, msg)
	}
}

func (c *Conn) writePump() {
	ticker := time.NewTicker(c.server.config.PingInterval)
	defer func() {
		ticker.Stop()
		c.Close()
	}()

	for {
		select {
		case msg, ok := <-c.sendCh:
			c.ws.SetWriteDeadline(time.Now().Add(c.server.config.WriteTimeout))
			if !ok {
				c.ws.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			data, err := c.server.config.Codec.Encode(msg)
			if err != nil {
				if c.server.config.OnError != nil {
					c.server.config.OnError(c, err)
				}
				continue
			}

			if err := c.ws.WriteMessage(websocket.TextMessage, data); err != nil {
				if c.server.config.OnError != nil {
					c.server.config.OnError(c, err)
				}
				return
			}

		case <-ticker.C:
			c.ws.SetWriteDeadline(time.Now().Add(c.server.config.WriteTimeout))
			if err := c.ws.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}

		case <-c.closeCh:
			return

		case <-c.ctx.Done():
			return
		}
	}
}
