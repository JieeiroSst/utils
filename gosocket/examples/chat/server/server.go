package main

import (
	"log"
	"time"

	"github.com/JIeeiroSst/utils/gosocket/core"
	"github.com/JIeeiroSst/utils/gosocket/helper"
	"github.com/JIeeiroSst/utils/gosocket/middleware"
	"github.com/JIeeiroSst/utils/gosocket/server"
)

type ChatMessage struct {
	UserID    string    `json:"user_id"`
	Username  string    `json:"username"`
	Message   string    `json:"message"`
	Room      string    `json:"room"`
	Timestamp time.Time `json:"timestamp"`
}

type JoinRoomMessage struct {
	Room     string `json:"room"`
	Username string `json:"username"`
}

func main() {
	srv := server.New(server.Config{
		Addr: ":8080",
		Path: "/ws",
		OnConnect: func(conn core.Connection) {
			log.Printf("Client connected: %s from %s", conn.ID(), conn.RemoteAddr())
		},
		OnDisconnect: func(conn core.Connection) {
			log.Printf("Client disconnected: %s", conn.ID())
		},
		OnError: func(conn core.Connection, err error) {
			log.Printf("Error from %s: %v", conn.ID(), err)
		},
	})

	srv.Use(middleware.Recover())
	srv.Use(middleware.Logger())

	srv.HandleFunc("join", func(ctx *core.HandlerContext) error {
		var msg JoinRoomMessage
		if err := ctx.Bind(&msg); err != nil {
			return err
		}

		ctx.Set("username", msg.Username)

		if err := helper.JoinRoom(ctx, msg.Room); err != nil {
			return err
		}

		return helper.BroadcastTo(ctx, msg.Room, "user_joined", map[string]interface{}{
			"username": msg.Username,
			"room":     msg.Room,
			"time":     time.Now(),
		})
	})

	srv.HandleFunc("leave", func(ctx *core.HandlerContext) error {
		var msg struct {
			Room string `json:"room"`
		}
		if err := ctx.Bind(&msg); err != nil {
			return err
		}

		username, _ := ctx.Get("username")

		if err := helper.LeaveRoom(ctx, msg.Room); err != nil {
			return err
		}

		return helper.BroadcastTo(ctx, msg.Room, "user_left", map[string]interface{}{
			"username": username,
			"room":     msg.Room,
			"time":     time.Now(),
		})
	})

	srv.HandleFunc("chat", func(ctx *core.HandlerContext) error {
		var msg ChatMessage
		if err := ctx.Bind(&msg); err != nil {
			return err
		}

		username, ok := ctx.Get("username")
		if !ok {
			username = "Anonymous"
		}

		msg.Username = username.(string)
		msg.Timestamp = time.Now()

		return helper.BroadcastTo(ctx, msg.Room, "chat", msg)
	})

	srv.HandleFunc("typing", func(ctx *core.HandlerContext) error {
		var msg struct {
			Room   string `json:"room"`
			Typing bool   `json:"typing"`
		}
		if err := ctx.Bind(&msg); err != nil {
			return err
		}

		username, _ := ctx.Get("username")

		return helper.BroadcastExcept(ctx, "typing", map[string]interface{}{
			"username": username,
			"room":     msg.Room,
			"typing":   msg.Typing,
		})
	})

	srv.HandleFunc("get_users", func(ctx *core.HandlerContext) error {
		var msg struct {
			Room string `json:"room"`
		}
		if err := ctx.Bind(&msg); err != nil {
			return err
		}

		s := ctx.Server().(*server.Server)
		room := s.GetRoom(msg.Room)

		connections := room.Connections()
		users := make([]map[string]interface{}, 0, len(connections))

		for _, conn := range connections {
			if username, ok := conn.Get("username"); ok {
				users = append(users, map[string]interface{}{
					"id":       conn.ID(),
					"username": username,
				})
			}
		}

		return ctx.Send("users_list", map[string]interface{}{
			"room":  msg.Room,
			"users": users,
			"count": len(users),
		})
	})

	log.Println("Starting chat server on :8080")
	if err := srv.Run(); err != nil {
		log.Fatal(err)
	}
}
