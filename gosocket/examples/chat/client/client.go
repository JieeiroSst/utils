package main

import (
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/JIeeiroSst/utils/gosocket/client"
	"github.com/JIeeiroSst/utils/gosocket/core"
)

type ChatMessage struct {
	Username  string    `json:"username"`
	Message   string    `json:"message"`
	Room      string    `json:"room"`
	Timestamp time.Time `json:"timestamp"`
}

func main() {
	c := client.New(client.Config{
		URL: "ws://localhost:8080/ws",
		OnConnect: func() {
			log.Println("Connected to server")
		},
		OnDisconnect: func() {
			log.Println("Disconnected from server")
		},
		OnError: func(err error) {
			log.Printf("Error: %v", err)
		},
		OnReconnecting: func(attempt int) {
			log.Printf("Reconnecting... attempt %d", attempt)
		},
		ReconnectInterval:    5 * time.Second,
		MaxReconnectAttempts: 10,
	})

	c.OnFunc("chat", func(ctx *core.HandlerContext) error {
		var msg ChatMessage
		if err := ctx.Bind(&msg); err != nil {
			return err
		}

		log.Printf("[%s] %s: %s", msg.Room, msg.Username, msg.Message)
		return nil
	})

	c.OnFunc("user_joined", func(ctx *core.HandlerContext) error {
		var data map[string]interface{}
		if err := ctx.Bind(&data); err != nil {
			return err
		}

		log.Printf("User joined: %v in room %v", data["username"], data["room"])
		return nil
	})

	c.OnFunc("user_left", func(ctx *core.HandlerContext) error {
		var data map[string]interface{}
		if err := ctx.Bind(&data); err != nil {
			return err
		}

		log.Printf("User left: %v from room %v", data["username"], data["room"])
		return nil
	})

	c.OnFunc("typing", func(ctx *core.HandlerContext) error {
		var data map[string]interface{}
		if err := ctx.Bind(&data); err != nil {
			return err
		}

		if data["typing"].(bool) {
			log.Printf("%v is typing...", data["username"])
		}
		return nil
	})

	if err := c.ConnectWithRetry(); err != nil {
		log.Fatal(err)
	}

	time.Sleep(time.Second) 
	c.Emit("join", map[string]interface{}{
		"room":     "general",
		"username": "Alice",
	})

	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		count := 0
		for range ticker.C {
			count++
			c.Emit("chat", map[string]interface{}{
				"room":    "general",
				"message": "Hello from client! " + time.Now().Format("15:04:05"),
			})

			if count >= 5 {
				return
			}
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	<-sigCh

	c.Emit("leave", map[string]interface{}{
		"room": "general",
	})

	time.Sleep(time.Second)
	c.Close()
	log.Println("Client closed")
}
