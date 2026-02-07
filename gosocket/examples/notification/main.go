package main

import (
	"log"
	"time"

	"github.com/JIeeiroSst/utils/gosocket/core"
	"github.com/JIeeiroSst/utils/gosocket/helper"
	"github.com/JIeeiroSst/utils/gosocket/middleware"
	"github.com/JIeeiroSst/utils/gosocket/server"
)

type Notification struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"`
	Title    string                 `json:"title"`
	Message  string                 `json:"message"`
	Data     map[string]interface{} `json:"data,omitempty"`
	Priority string                 `json:"priority"`
	Read     bool                   `json:"read"`
	Created  time.Time              `json:"created"`
}

func main() {
	srv := server.New(server.Config{
		Addr: ":8080",
		Path: "/ws",
		OnConnect: func(conn core.Connection) {
			log.Printf("Client connected: %s", conn.ID())
		},
	})

	srv.Use(middleware.Logger())

	srv.Use(&middleware.AuthMiddleware{
		Validate: func(ctx *core.HandlerContext) (string, error) {
			token, ok := ctx.Message.Data.(map[string]interface{})["token"]
			if !ok {
				return "", &AuthError{Code: "UNAUTHORIZED", Message: "Token required"}
			}

			userID := validateToken(token.(string))
			if userID == "" {
				return "", &AuthError{Code: "INVALID_TOKEN", Message: "Invalid token"}
			}

			return userID, nil
		},
	})

	srv.HandleFunc("subscribe", func(ctx *core.HandlerContext) error {
		userID, ok := helper.GetUserID(ctx)
		if !ok {
			return &AuthError{Code: "UNAUTHORIZED", Message: "Unauthorized"}
		}

		roomName := "user:" + userID
		if err := helper.JoinRoom(ctx, roomName); err != nil {
			return err
		}

		log.Printf("User %s subscribed to notifications", userID)

		return ctx.Send("subscribed", map[string]interface{}{
			"user_id": userID,
			"status":  "success",
		})
	})

	srv.HandleFunc("mark_read", func(ctx *core.HandlerContext) error {
		var msg struct {
			NotificationID string `json:"notification_id"`
		}
		if err := ctx.Bind(&msg); err != nil {
			return err
		}

		userID, _ := helper.GetUserID(ctx)

		log.Printf("User %s marked notification %s as read", userID, msg.NotificationID)

		return ctx.Send("notification_read", map[string]interface{}{
			"notification_id": msg.NotificationID,
			"status":          "success",
		})
	})

	srv.HandleFunc("get_unread_count", func(ctx *core.HandlerContext) error {
		userID, _ := helper.GetUserID(ctx)

		count := getUnreadCount(userID)

		return ctx.Send("unread_count", map[string]interface{}{
			"count": count,
		})
	})

	srv.HandleFunc("get_notifications", func(ctx *core.HandlerContext) error {
		var msg struct {
			Limit  int  `json:"limit"`
			Offset int  `json:"offset"`
			Unread bool `json:"unread"`
		}
		ctx.Bind(&msg)

		if msg.Limit == 0 {
			msg.Limit = 20
		}

		userID, _ := helper.GetUserID(ctx)
		notifications := getNotifications(userID, msg.Limit, msg.Offset, msg.Unread)

		return ctx.Send("notifications", map[string]interface{}{
			"items": notifications,
			"total": len(notifications),
		})
	})

	srv.HandleFunc("mark_all_read", func(ctx *core.HandlerContext) error {
		userID, _ := helper.GetUserID(ctx)

		count := markAllRead(userID)

		return ctx.Send("all_marked_read", map[string]interface{}{
			"count": count,
		})
	})

	srv.HandleFunc("delete_notification", func(ctx *core.HandlerContext) error {
		var msg struct {
			NotificationID string `json:"notification_id"`
		}
		if err := ctx.Bind(&msg); err != nil {
			return err
		}

		userID, _ := helper.GetUserID(ctx)

		if err := deleteNotification(userID, msg.NotificationID); err != nil {
			return err
		}

		return ctx.Send("notification_deleted", map[string]interface{}{
			"notification_id": msg.NotificationID,
			"status":          "success",
		})
	})

	go simulateNotifications()

	log.Println("Starting notification server on :8080")
	if err := srv.Run(); err != nil {
		log.Fatal(err)
	}
}

func simulateNotifications() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		notification := Notification{
			ID:       generateID(),
			Type:     "info",
			Title:    "New Update",
			Message:  "You have a new update available",
			Priority: "normal",
			Created:  time.Now(),
		}

		log.Println("Sent notification to user123")
		_ = notification
	}
}

func validateToken(token string) string {
	if token == "valid_token" {
		return "user123"
	}
	return ""
}

func getUnreadCount(userID string) int {
	return 5
}

func getNotifications(userID string, limit, offset int, unread bool) []Notification {
	notifications := []Notification{
		{
			ID:       "notif_1",
			Type:     "info",
			Title:    "Welcome",
			Message:  "Welcome to the platform",
			Priority: "normal",
			Read:     false,
			Created:  time.Now().Add(-1 * time.Hour),
		},
		{
			ID:       "notif_2",
			Type:     "alert",
			Title:    "Security Alert",
			Message:  "New login detected",
			Priority: "high",
			Read:     false,
			Created:  time.Now().Add(-30 * time.Minute),
		},
	}

	if unread {
		filtered := []Notification{}
		for _, n := range notifications {
			if !n.Read {
				filtered = append(filtered, n)
			}
		}
		return filtered
	}

	return notifications
}

func markAllRead(userID string) int {
	return 5
}

func deleteNotification(userID, notificationID string) error {
	log.Printf("Deleted notification %s for user %s", notificationID, userID)
	return nil
}

func generateID() string {
	return time.Now().Format("20060102150405")
}

type AuthError struct {
	Code    string
	Message string
}

func (e *AuthError) Error() string {
	return e.Message
}
