package helper

import (
	"github.com/JIeeiroSst/utils/gosocket/core"
	"github.com/JIeeiroSst/utils/gosocket/server"
)

type BroadcastHelper struct {
	server *server.Server
}

func NewBroadcastHelper(srv *server.Server) *BroadcastHelper {
	return &BroadcastHelper{server: srv}
}

func Broadcast(ctx *core.HandlerContext, event string, data interface{}) error {
	srv, ok := ctx.Server().(*server.Server)
	if !ok {
		return ErrInvalidServer
	}

	msg := &core.Message{
		Event: event,
		Data:  data,
	}

	return srv.Broadcast(msg)
}

func BroadcastTo(ctx *core.HandlerContext, room string, event string, data interface{}) error {
	srv, ok := ctx.Server().(*server.Server)
	if !ok {
		return ErrInvalidServer
	}

	msg := &core.Message{
		Event: event,
		Data:  data,
	}

	return srv.BroadcastTo(room, msg)
}

func BroadcastExcept(ctx *core.HandlerContext, event string, data interface{}) error {
	srv, ok := ctx.Server().(*server.Server)
	if !ok {
		return ErrInvalidServer
	}

	msg := &core.Message{
		Event: event,
		Data:  data,
	}

	return srv.BroadcastExcept(ctx.Conn.ID(), msg)
}

func JoinRoom(ctx *core.HandlerContext, roomName string) error {
	srv, ok := ctx.Server().(*server.Server)
	if !ok {
		return ErrInvalidServer
	}

	room := srv.GetRoom(roomName)
	return room.Join(ctx.Conn)
}

func LeaveRoom(ctx *core.HandlerContext, roomName string) error {
	srv, ok := ctx.Server().(*server.Server)
	if !ok {
		return ErrInvalidServer
	}

	room := srv.GetRoom(roomName)
	return room.Leave(ctx.Conn)
}

func GetUserID(ctx *core.HandlerContext) (string, bool) {
	val, ok := ctx.Get("user_id")
	if !ok {
		return "", false
	}

	userID, ok := val.(string)
	return userID, ok
}

func SetUserID(ctx *core.HandlerContext, userID string) {
	ctx.Set("user_id", userID)
}

var ErrInvalidServer = &Error{Code: "INVALID_SERVER", Message: "Invalid server instance"}

type Error struct {
	Code    string
	Message string
}

func (e *Error) Error() string {
	return e.Message
}
