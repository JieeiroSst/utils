package circuit_breaker

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/JIeeiroSst/utils/logger"
	"github.com/sony/gobreaker"
	"go.uber.org/zap"
)

type ClientCircuitBreakerProxy struct {
	logger *zap.Logger
	gb     *gobreaker.CircuitBreaker
}

func shouldBeSwitchedToOpen(counts gobreaker.Counts) bool {
	failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
	return counts.Requests >= 3 && failureRatio >= 0.6
}

func NewClientCircuitBreakerProxy() *ClientCircuitBreakerProxy {
	logger := logger.ConfigZap(context.Background())

	cfg := gobreaker.Settings{
		Interval:    5 * time.Second,
		Timeout:     7 * time.Second,
		ReadyToTrip: shouldBeSwitchedToOpen,
		OnStateChange: func(_ string, from gobreaker.State, to gobreaker.State) {
			logger.Sugar().Infof("state changed from %v %v", from, to)
		},
	}

	return &ClientCircuitBreakerProxy{
		logger: logger,
		gb:     gobreaker.NewCircuitBreaker(cfg),
	}
}

func (c *ClientCircuitBreakerProxy) Send(endpoint string) (interface{}, error) {
	data, err := c.gb.Execute(func() (interface{}, error) {
		resp, err := http.Get(endpoint)
		if err != nil {
			c.logger.Sugar().Error(err)
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			c.logger.Sugar().Error(err)
		}
		return string(body), err
	})
	if err != nil {
		c.logger.Sugar().Error(err)
	}
	return data, nil
}
