package http

import (
	"bytes"
	"context"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/sony/gobreaker"
)

type clientCircuitBreakerProxy struct {
	gb *gobreaker.CircuitBreaker
}

type ClientCircuitBreakerProxy interface {
	Get(ctx context.Context, url string) (interface{}, error)
	Post(ctx context.Context, url, method string, data []byte) (interface{}, error)
}

func shouldBeSwitchedToOpen(counts gobreaker.Counts) bool {
	failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
	return counts.Requests >= 3 && failureRatio >= 0.6
}

func NewClientCircuitBreakerProxy() ClientCircuitBreakerProxy {
	logger := log.New(os.Stdout, "CB\t", log.LstdFlags)

	cfg := gobreaker.Settings{
		Interval:    5 * time.Second,
		Timeout:     7 * time.Second,
		ReadyToTrip: shouldBeSwitchedToOpen,
		OnStateChange: func(_ string, from gobreaker.State, to gobreaker.State) {
			logger.Println("state changed from", from.String(), "to", to.String())
		},
	}

	return &clientCircuitBreakerProxy{
		gb: gobreaker.NewCircuitBreaker(cfg),
	}
}

func (c *clientCircuitBreakerProxy) Get(ctx context.Context, url string) (interface{}, error) {
	data, err := c.gb.Execute(func() (interface{}, error) {
		resp, err := http.Get(url)
		if err != nil {
			return nil, err
		}

		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		return body, nil
	})
	if err != nil {
		return nil, err
	}
	return data.([]byte), nil
}

func (c *clientCircuitBreakerProxy) Post(ctx context.Context, url, method string, paramater []byte) (interface{}, error) {
	data, err := c.gb.Execute(func() (interface{}, error) {
		req, err := http.NewRequest(method, url, bytes.NewBuffer(paramater))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		return body, err
	})
	if err != nil {
		return nil, err
	}
	return data.([]byte), nil
}
