package core

import (
	"context"
	"time"
)

type Message struct {
	Event     string      `json:"event"`
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
	ID        string      `json:"id,omitempty"`
}

type Connection interface {
	ID() string
	Send(msg *Message) error
	Close() error
	Context() context.Context
	Set(key string, value interface{})
	Get(key string) (interface{}, bool)
	RemoteAddr() string
}

type Handler interface {
	Handle(ctx *HandlerContext) error
}

type HandlerFunc func(ctx *HandlerContext) error

func (f HandlerFunc) Handle(ctx *HandlerContext) error {
	return f(ctx)
}

type Codec interface {
	Encode(msg *Message) ([]byte, error)
	Decode(data []byte) (*Message, error)
	Name() string
}

type Middleware interface {
	Process(ctx *HandlerContext, next Handler) error
}

type MiddlewareFunc func(ctx *HandlerContext, next Handler) error

func (f MiddlewareFunc) Process(ctx *HandlerContext, next Handler) error {
	return f(ctx, next)
}

type Broadcaster interface {
	Broadcast(msg *Message) error
	BroadcastTo(room string, msg *Message) error
	BroadcastExcept(excludeID string, msg *Message) error
}

type Room interface {
	Join(conn Connection) error
	Leave(conn Connection) error
	Broadcast(msg *Message) error
	Connections() []Connection
	Count() int
}

type EventEmitter interface {
	Emit(event string, data interface{}) error
	On(event string, handler Handler)
	Off(event string)
}
