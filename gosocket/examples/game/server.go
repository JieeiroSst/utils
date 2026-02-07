package main

import (
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/JIeeiroSst/utils/gosocket/core"
	"github.com/JIeeiroSst/utils/gosocket/helper"
	"github.com/JIeeiroSst/utils/gosocket/middleware"
	"github.com/JIeeiroSst/utils/gosocket/server"
)

type GameState struct {
	ID      string             `json:"id"`
	Players map[string]*Player `json:"players"`
	Status  string             `json:"status"` 
	mu      sync.RWMutex
}

type Player struct {
	ID       string  `json:"id"`
	Username string  `json:"username"`
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
	Score    int     `json:"score"`
	Ready    bool    `json:"ready"`
}

type PlayerAction struct {
	Action string  `json:"action"` 
	X      float64 `json:"x,omitempty"`
	Y      float64 `json:"y,omitempty"`
	Target string  `json:"target,omitempty"`
}

var (
	games   = make(map[string]*GameState)
	gamesMu sync.RWMutex
)

func main() {
	srv := server.New(server.Config{
		Addr: ":8080",
		Path: "/ws",
		OnConnect: func(conn core.Connection) {
			log.Printf("Player connected: %s", conn.ID())
		},
		OnDisconnect: func(conn core.Connection) {
			playerID := conn.ID()
			log.Printf("Player disconnected: %s", playerID)

			gamesMu.Lock()
			for _, game := range games {
				game.mu.Lock()
				if _, exists := game.Players[playerID]; exists {
					delete(game.Players, playerID)
					log.Printf("Removed player %s from game %s", playerID, game.ID)
				}
				game.mu.Unlock()
			}
			gamesMu.Unlock()
		},
	})

	srv.Use(middleware.Logger())
	srv.Use(middleware.Recover())

	srv.HandleFunc("create_game", func(ctx *core.HandlerContext) error {
		var msg struct {
			Username string `json:"username"`
		}
		if err := ctx.Bind(&msg); err != nil {
			return err
		}

		gameID := generateGameID()
		game := &GameState{
			ID:      gameID,
			Players: make(map[string]*Player),
			Status:  "waiting",
		}

		player := &Player{
			ID:       ctx.Conn.ID(),
			Username: msg.Username,
			X:        rand.Float64() * 100,
			Y:        rand.Float64() * 100,
			Score:    0,
			Ready:    false,
		}

		game.Players[player.ID] = player

		gamesMu.Lock()
		games[gameID] = game
		gamesMu.Unlock()

		helper.JoinRoom(ctx, "game:"+gameID)

		ctx.Set("game_id", gameID)
		ctx.Set("username", msg.Username)

		log.Printf("Game %s created by %s", gameID, msg.Username)

		return ctx.Send("game_created", map[string]interface{}{
			"game_id": gameID,
			"player":  player,
		})
	})

	srv.HandleFunc("join_game", func(ctx *core.HandlerContext) error {
		var msg struct {
			GameID   string `json:"game_id"`
			Username string `json:"username"`
		}
		if err := ctx.Bind(&msg); err != nil {
			return err
		}

		gamesMu.RLock()
		game, ok := games[msg.GameID]
		gamesMu.RUnlock()

		if !ok {
			return &GameError{Code: "GAME_NOT_FOUND", Message: "Game not found"}
		}

		game.mu.RLock()
		playerCount := len(game.Players)
		game.mu.RUnlock()

		if playerCount >= 4 {
			return &GameError{Code: "GAME_FULL", Message: "Game is full"}
		}

		player := &Player{
			ID:       ctx.Conn.ID(),
			Username: msg.Username,
			X:        rand.Float64() * 100,
			Y:        rand.Float64() * 100,
			Score:    0,
			Ready:    false,
		}

		game.mu.Lock()
		game.Players[player.ID] = player
		game.mu.Unlock()

		helper.JoinRoom(ctx, "game:"+msg.GameID)

		ctx.Set("game_id", msg.GameID)
		ctx.Set("username", msg.Username)

		log.Printf("Player %s joined game %s", msg.Username, msg.GameID)

		helper.BroadcastTo(ctx, "game:"+msg.GameID, "player_joined", map[string]interface{}{
			"player": player,
		})

		return ctx.Send("game_joined", map[string]interface{}{
			"game_id": msg.GameID,
			"player":  player,
		})
	})

	srv.HandleFunc("ready", func(ctx *core.HandlerContext) error {
		gameID, ok := ctx.Get("game_id")
		if !ok {
			return &GameError{Code: "NOT_IN_GAME", Message: "Not in any game"}
		}

		gamesMu.RLock()
		game := games[gameID.(string)]
		gamesMu.RUnlock()

		playerID := ctx.Conn.ID()

		game.mu.Lock()
		if player, ok := game.Players[playerID]; ok {
			player.Ready = true
		}

		allReady := true
		for _, p := range game.Players {
			if !p.Ready {
				allReady = false
				break
			}
		}

		if allReady && len(game.Players) >= 2 {
			game.Status = "playing"
			log.Printf("Game %s started with %d players", gameID, len(game.Players))
		}
		game.mu.Unlock()

		return helper.BroadcastTo(ctx, "game:"+gameID.(string), "game_state", map[string]interface{}{
			"status":       game.Status,
			"player_count": len(game.Players),
		})
	})

	srv.HandleFunc("action", func(ctx *core.HandlerContext) error {
		var action PlayerAction
		if err := ctx.Bind(&action); err != nil {
			return err
		}

		gameID, ok := ctx.Get("game_id")
		if !ok {
			return &GameError{Code: "NOT_IN_GAME", Message: "Not in any game"}
		}

		gamesMu.RLock()
		game := games[gameID.(string)]
		gamesMu.RUnlock()

		playerID := ctx.Conn.ID()

		game.mu.Lock()
		player, ok := game.Players[playerID]
		if !ok {
			game.mu.Unlock()
			return &GameError{Code: "PLAYER_NOT_FOUND", Message: "Player not found"}
		}

		switch action.Action {
		case "move":
			player.X = action.X
			player.Y = action.Y
			log.Printf("Player %s moved to (%.2f, %.2f)", playerID, action.X, action.Y)
		case "shoot":
			log.Printf("Player %s shoots", playerID)
			if action.Target != "" {
				if target, exists := game.Players[action.Target]; exists {
					target.Score--
					player.Score++
					log.Printf("Player %s hit %s", playerID, action.Target)
				}
			}
		case "jump":
			log.Printf("Player %s jumps", playerID)
		}
		game.mu.Unlock()

		return helper.BroadcastTo(ctx, "game:"+gameID.(string), "player_action", map[string]interface{}{
			"player_id": playerID,
			"action":    action,
			"player":    player,
		})
	})

	srv.HandleFunc("get_state", func(ctx *core.HandlerContext) error {
		gameID, ok := ctx.Get("game_id")
		if !ok {
			return &GameError{Code: "NOT_IN_GAME", Message: "Not in any game"}
		}

		gamesMu.RLock()
		game := games[gameID.(string)]
		gamesMu.RUnlock()

		game.mu.RLock()
		players := make([]*Player, 0, len(game.Players))
		for _, p := range game.Players {
			players = append(players, p)
		}
		game.mu.RUnlock()

		return ctx.Send("game_state", map[string]interface{}{
			"game_id": game.ID,
			"status":  game.Status,
			"players": players,
		})
	})

	srv.HandleFunc("leave_game", func(ctx *core.HandlerContext) error {
		gameID, ok := ctx.Get("game_id")
		if !ok {
			return nil
		}

		gamesMu.RLock()
		game := games[gameID.(string)]
		gamesMu.RUnlock()

		playerID := ctx.Conn.ID()

		game.mu.Lock()
		delete(game.Players, playerID)
		remainingPlayers := len(game.Players)
		game.mu.Unlock()

		helper.LeaveRoom(ctx, "game:"+gameID.(string))

		log.Printf("Player %s left game %s (%d players remaining)", playerID, gameID, remainingPlayers)

		return helper.BroadcastTo(ctx, "game:"+gameID.(string), "player_left", map[string]interface{}{
			"player_id": playerID,
		})
	})

	log.Println("Starting game server on :8080")
	log.Println("Supported events: create_game, join_game, ready, action, get_state, leave_game")

	if err := srv.Run(); err != nil {
		log.Fatal(err)
	}
}

func generateGameID() string {
	return "game_" + time.Now().Format("20060102150405")
}

type GameError struct {
	Code    string
	Message string
}

func (e *GameError) Error() string {
	return e.Message
}
