package server

import (
	"context"
	"errors"
	"log"
	"net/http"
	"sync"

	"github.com/JIeeiroSst/utils/gosocket/core"
	"github.com/gorilla/websocket"
)

var (
	ErrConnectionClosed = errors.New("connection closed")
	ErrHandlerNotFound  = errors.New("handler not found")
)

// Server WebSocket server
type Server struct {
	config      Config
	upgrader    websocket.Upgrader
	handlers    map[string]core.Handler
	handlersMu  sync.RWMutex
	middlewares []core.Middleware
	connections map[string]*Conn
	connsMu     sync.RWMutex
	rooms       map[string]*Room
	roomsMu     sync.RWMutex
	register    chan *Conn
	unregister  chan *Conn
	broadcast   chan *core.Message
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
}

func New(cfg Config) *Server {
	cfg = cfg.Merge()
	ctx, cancel := context.WithCancel(context.Background())

	s := &Server{
		config:      cfg,
		handlers:    make(map[string]core.Handler),
		middlewares: make([]core.Middleware, 0),
		connections: make(map[string]*Conn),
		rooms:       make(map[string]*Room),
		register:    make(chan *Conn),
		unregister:  make(chan *Conn),
		broadcast:   make(chan *core.Message),
		ctx:         ctx,
		cancel:      cancel,
		upgrader: websocket.Upgrader{
			ReadBufferSize:   cfg.ReadBufferSize,
			WriteBufferSize:  cfg.WriteBufferSize,
			HandshakeTimeout: cfg.HandshakeTimeout,
			CheckOrigin: func(r *http.Request) bool {
				return cfg.CheckOrigin(r)
			},
		},
	}

	return s
}

func (s *Server) Handle(event string, handler core.Handler) {
	s.handlersMu.Lock()
	defer s.handlersMu.Unlock()
	s.handlers[event] = handler
}

func (s *Server) HandleFunc(event string, fn func(ctx *core.HandlerContext) error) {
	s.Handle(event, core.HandlerFunc(fn))
}

func (s *Server) Use(middleware core.Middleware) {
	s.middlewares = append(s.middlewares, middleware)
}

func (s *Server) UseFunc(fn func(ctx *core.HandlerContext, next core.Handler) error) {
	s.Use(core.MiddlewareFunc(fn))
}

func (s *Server) Broadcast(msg *core.Message) error {
	select {
	case s.broadcast <- msg:
		return nil
	case <-s.ctx.Done():
		return s.ctx.Err()
	}
}

func (s *Server) BroadcastTo(roomName string, msg *core.Message) error {
	s.roomsMu.RLock()
	room, ok := s.rooms[roomName]
	s.roomsMu.RUnlock()

	if !ok {
		return errors.New("room not found")
	}

	return room.Broadcast(msg)
}

func (s *Server) BroadcastExcept(excludeID string, msg *core.Message) error {
	s.connsMu.RLock()
	defer s.connsMu.RUnlock()

	for id, conn := range s.connections {
		if id != excludeID {
			conn.Send(msg)
		}
	}

	return nil
}

func (s *Server) GetConnection(id string) (core.Connection, bool) {
	s.connsMu.RLock()
	defer s.connsMu.RUnlock()
	conn, ok := s.connections[id]
	return conn, ok
}

func (s *Server) GetRoom(name string) core.Room {
	s.roomsMu.Lock()
	defer s.roomsMu.Unlock()

	room, ok := s.rooms[name]
	if !ok {
		room = NewRoom(name)
		s.rooms[name] = room
	}

	return room
}

func (s *Server) Run() error {
	s.wg.Add(1)
	go s.hub()

	http.HandleFunc(s.config.Path, s.handleWebSocket)

	log.Printf("WebSocket server listening on %s%s", s.config.Addr, s.config.Path)

	return http.ListenAndServe(s.config.Addr, nil)
}

func (s *Server) Shutdown() error {
	s.cancel()

	s.connsMu.Lock()
	for _, conn := range s.connections {
		conn.Close()
	}
	s.connsMu.Unlock()

	s.wg.Wait()
	return nil
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	ws, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Upgrade error: %v", err)
		return
	}

	conn := NewConn(ws, s)
	s.register <- conn

	go conn.writePump()
	go conn.readPump()
}

func (s *Server) hub() {
	defer s.wg.Done()

	for {
		select {
		case conn := <-s.register:
			s.connsMu.Lock()
			s.connections[conn.ID()] = conn
			s.connsMu.Unlock()

			if s.config.OnConnect != nil {
				s.config.OnConnect(conn)
			}

		case conn := <-s.unregister:
			s.connsMu.Lock()
			delete(s.connections, conn.ID())
			s.connsMu.Unlock()

			s.roomsMu.RLock()
			for _, room := range s.rooms {
				room.Leave(conn)
			}
			s.roomsMu.RUnlock()

			if s.config.OnDisconnect != nil {
				s.config.OnDisconnect(conn)
			}

		case msg := <-s.broadcast:
			s.connsMu.RLock()
			for _, conn := range s.connections {
				conn.Send(msg)
			}
			s.connsMu.RUnlock()

		case <-s.ctx.Done():
			return
		}
	}
}

func (s *Server) handleMessage(conn *Conn, msg *core.Message) {
	s.handlersMu.RLock()
	handler, ok := s.handlers[msg.Event]
	s.handlersMu.RUnlock()

	if !ok {
		if s.config.OnError != nil {
			s.config.OnError(conn, ErrHandlerNotFound)
		}
		return
	}

	ctx := core.NewHandlerContext(conn.Context(), conn, msg)
	ctx.SetServer(s)

	finalHandler := handler
	for i := len(s.middlewares) - 1; i >= 0; i-- {
		middleware := s.middlewares[i]
		prevHandler := finalHandler
		finalHandler = core.HandlerFunc(func(ctx *core.HandlerContext) error {
			return middleware.Process(ctx, prevHandler)
		})
	}

	if err := finalHandler.Handle(ctx); err != nil {
		if s.config.OnError != nil {
			s.config.OnError(conn, err)
		}
	}
}
