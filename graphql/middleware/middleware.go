package middleware

import (
	"context"
	"fmt"
	"log"
	"time"

	gqlcontext "github.com/JIeeiroSst/utils/graphql/context"
	"github.com/JIeeiroSst/utils/graphql/errors"
)

type Middleware func(context.Context, func(context.Context) (interface{}, error)) (interface{}, error)

func Chain(middlewares ...Middleware) Middleware {
	return func(ctx context.Context, next func(context.Context) (interface{}, error)) (interface{}, error) {
		handler := next
		for i := len(middlewares) - 1; i >= 0; i-- {
			currentMiddleware := middlewares[i]
			currentHandler := handler
			handler = func(ctx context.Context) (interface{}, error) {
				return currentMiddleware(ctx, currentHandler)
			}
		}
		return handler(ctx)
	}
}

func AuthMiddleware(required bool) Middleware {
	return func(ctx context.Context, next func(context.Context) (interface{}, error)) (interface{}, error) {
		if required && !gqlcontext.IsAuthenticated(ctx) {
			return nil, errors.Unauthorized("Authentication required")
		}
		return next(ctx)
	}
}

func RoleMiddleware(allowedRoles ...string) Middleware {
	return func(ctx context.Context, next func(context.Context) (interface{}, error)) (interface{}, error) {
		if !gqlcontext.IsAuthenticated(ctx) {
			return nil, errors.Unauthorized("Authentication required")
		}

		if !gqlcontext.HasAnyRole(ctx, allowedRoles...) {
			return nil, errors.Forbidden("Insufficient permissions")
		}

		return next(ctx)
	}
}

func LoggingMiddleware(logger *log.Logger) Middleware {
	return func(ctx context.Context, next func(context.Context) (interface{}, error)) (interface{}, error) {
		start := time.Now()

		requestID, _ := gqlcontext.GetRequestID(ctx)
		userID, _ := gqlcontext.GetUserID(ctx)

		logger.Printf("[%s] Starting request - User: %s", requestID, userID)

		result, err := next(ctx)

		duration := time.Since(start)
		if err != nil {
			logger.Printf("[%s] Request failed - Duration: %v - Error: %v", requestID, duration, err)
		} else {
			logger.Printf("[%s] Request succeeded - Duration: %v", requestID, duration)
		}

		return result, err
	}
}

func RecoveryMiddleware() Middleware {
	return func(ctx context.Context, next func(context.Context) (interface{}, error)) (result interface{}, err error) {
		defer func() {
			if r := recover(); r != nil {
				err = errors.Internal(fmt.Sprintf("panic recovered: %v", r))
			}
		}()

		return next(ctx)
	}
}

func TimeoutMiddleware(timeout time.Duration) Middleware {
	return func(ctx context.Context, next func(context.Context) (interface{}, error)) (interface{}, error) {
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		type result struct {
			data interface{}
			err  error
		}

		resultChan := make(chan result, 1)

		go func() {
			data, err := next(ctx)
			resultChan <- result{data: data, err: err}
		}()

		select {
		case res := <-resultChan:
			return res.data, res.err
		case <-ctx.Done():
			return nil, errors.Internal("Request timeout")
		}
	}
}

type Validator interface {
	Validate() error
}

func ValidateMiddleware() Middleware {
	return func(ctx context.Context, next func(context.Context) (interface{}, error)) (interface{}, error) {
		return next(ctx)
	}
}

type RateLimiter interface {
	Allow(ctx context.Context, key string) (bool, error)
}

func RateLimitMiddleware(limiter RateLimiter) Middleware {
	return func(ctx context.Context, next func(context.Context) (interface{}, error)) (interface{}, error) {
		userID, ok := gqlcontext.GetUserID(ctx)
		if !ok {
			userID = "anonymous"
		}

		allowed, err := limiter.Allow(ctx, userID)
		if err != nil {
			return nil, errors.Internal("Rate limit check failed")
		}

		if !allowed {
			return nil, errors.RateLimit("Too many requests")
		}

		return next(ctx)
	}
}

func TenantMiddleware() Middleware {
	return func(ctx context.Context, next func(context.Context) (interface{}, error)) (interface{}, error) {
		tenantID, ok := gqlcontext.GetTenantID(ctx)
		if !ok {
			return nil, errors.BadRequest("Tenant ID required")
		}

		if tenantID == "" {
			return nil, errors.BadRequest("Invalid tenant ID")
		}

		return next(ctx)
	}
}
