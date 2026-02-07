package middleware

import (
	"log"
	"sync"
	"time"

	"github.com/JIeeiroSst/utils/gosocket/core"
)

var (
	ErrUnauthorized      = &Error{Code: "UNAUTHORIZED", Message: "Unauthorized"}
	ErrRateLimitExceeded = &Error{Code: "RATE_LIMIT_EXCEEDED", Message: "Rate limit exceeded"}
)

func Logger() core.Middleware {
	return core.MiddlewareFunc(func(ctx *core.HandlerContext, next core.Handler) error {
		start := time.Now()

		log.Printf("[%s] Event: %s, From: %s",
			ctx.Conn.ID(),
			ctx.Message.Event,
			ctx.Conn.RemoteAddr(),
		)

		err := next.Handle(ctx)

		log.Printf("[%s] Event: %s, Duration: %v, Error: %v",
			ctx.Conn.ID(),
			ctx.Message.Event,
			time.Since(start),
			err,
		)

		return err
	})
}

func Recover() core.Middleware {
	return core.MiddlewareFunc(func(ctx *core.HandlerContext, next core.Handler) error {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[%s] Panic recovered: %v", ctx.Conn.ID(), r)
			}
		}()

		return next.Handle(ctx)
	})
}

type AuthMiddleware struct {
	Validate func(ctx *core.HandlerContext) (string, error)
}

func (m *AuthMiddleware) Process(ctx *core.HandlerContext, next core.Handler) error {
	if _, ok := ctx.Get("user_id"); ok {
		return next.Handle(ctx)
	}

	userID, err := m.Validate(ctx)
	if err != nil {
		return err
	}

	ctx.Set("user_id", userID)

	return next.Handle(ctx)
}

type RateLimitMiddleware struct {
	requests map[string][]time.Time
	limit    int
	window   time.Duration
	mu       sync.Mutex
}

func NewRateLimit(limit int, window time.Duration) *RateLimitMiddleware {
	return &RateLimitMiddleware{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}
}

func (m *RateLimitMiddleware) Process(ctx *core.HandlerContext, next core.Handler) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	connID := ctx.Conn.ID()
	now := time.Now()

	requests := m.requests[connID]
	validRequests := make([]time.Time, 0)
	for _, t := range requests {
		if now.Sub(t) < m.window {
			validRequests = append(validRequests, t)
		}
	}

	if len(validRequests) >= m.limit {
		return ErrRateLimitExceeded
	}

	validRequests = append(validRequests, now)
	m.requests[connID] = validRequests

	return next.Handle(ctx)
}

type Error struct {
	Code    string
	Message string
}

func (e *Error) Error() string {
	return e.Message
}
