package circuit_breaker

import (
	"io"
	"net/http"
	"time"

	"github.com/JIeeiroSst/utils/logger"
	"github.com/sony/gobreaker"
	"go.uber.org/zap"
)

type ClientCircuitBreakerProxy struct {
	logger *zap.SugaredLogger
	gb     *gobreaker.CircuitBreaker
}

func shouldBeSwitchedToOpen(counts gobreaker.Counts) bool {
	failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
	return counts.Requests >= 3 && failureRatio >= 0.6
}

func NewClientCircuitBreakerProxy() *ClientCircuitBreakerProxy {
	logger := logger.ConfigZap()

	cfg := gobreaker.Settings{
		Interval:    5 * time.Second,
		Timeout:     7 * time.Second,
		ReadyToTrip: shouldBeSwitchedToOpen,
		OnStateChange: func(_ string, from gobreaker.State, to gobreaker.State) {
			logger.Info("state changed from", from.String(), "to", to.String())
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
			c.logger.Error(err)
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			c.logger.Error(err)
		}
		return string(body), err
	})
	if err != nil {
		c.logger.Error(err)
	}
	return data, nil
}
