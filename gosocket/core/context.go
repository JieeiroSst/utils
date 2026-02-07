package core

import (
	"context"
	"encoding/json"
)

type HandlerContext struct {
	ctx     context.Context
	Conn    Connection
	Message *Message
	server  interface{} 
}

func NewHandlerContext(ctx context.Context, conn Connection, msg *Message) *HandlerContext {
	return &HandlerContext{
		ctx:     ctx,
		Conn:    conn,
		Message: msg,
	}
}

func (c *HandlerContext) Context() context.Context {
	return c.ctx
}

func (c *HandlerContext) Bind(v interface{}) error {
	data, err := json.Marshal(c.Message.Data)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

func (c *HandlerContext) Send(event string, data interface{}) error {
	msg := &Message{
		Event: event,
		Data:  data,
	}
	return c.Conn.Send(msg)
}

func (c *HandlerContext) SetServer(server interface{}) {
	c.server = server
}

func (c *HandlerContext) Server() interface{} {
	return c.server
}

func (c *HandlerContext) Get(key string) (interface{}, bool) {
	return c.Conn.Get(key)
}

func (c *HandlerContext) Set(key string, value interface{}) {
	c.Conn.Set(key, value)
}