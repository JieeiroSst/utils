package idempotency

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

type Status string

const (
	StatusProcessing Status = "processing"
	StatusDone       Status = "done"
)

var (
	ErrDuplicate  = errors.New("message already processed successfully")
	ErrProcessing = errors.New("message is currently being processed")
)

type IdempotencyGuard struct {
	rdb           *redis.Client
	processingTTL time.Duration // TTL cho trạng thái "processing" (timeout nếu crash)
	doneTTL       time.Duration // TTL cho trạng thái "done"
}

func NewIdempotencyGuard(rdb *redis.Client) *IdempotencyGuard {
	return &IdempotencyGuard{
		rdb:           rdb,
		processingTTL: 30 * time.Second, // nếu crash, tự unlock sau 30s
		doneTTL:       24 * time.Hour,
	}
}

func (g *IdempotencyGuard) key(messageID string) string {
	return "msg:idempotency:" + messageID
}

// Acquire: gọi trước khi xử lý
// - Trả về nil nếu được phép xử lý
// - Trả về ErrDuplicate nếu đã done
// - Trả về ErrProcessing nếu đang có goroutine khác xử lý
func (g *IdempotencyGuard) Acquire(ctx context.Context, messageID string) error {
	key := g.key(messageID)

	// Dùng SET NX để atomic check-and-set
	ok, err := g.rdb.SetNX(ctx, key, string(StatusProcessing), g.processingTTL).Result()
	if err != nil {
		return err
	}

	if ok {
		// Không có key trước đó → được phép xử lý
		return nil
	}

	// Key đã tồn tại → check trạng thái
	val, err := g.rdb.Get(ctx, key).Result()
	if err != nil {
		return err
	}

	switch Status(val) {
	case StatusDone:
		return ErrDuplicate
	case StatusProcessing:
		return ErrProcessing
	default:
		return nil
	}
}

// MarkDone: gọi sau khi xử lý THÀNH CÔNG
func (g *IdempotencyGuard) MarkDone(ctx context.Context, messageID string) error {
	key := g.key(messageID)
	return g.rdb.Set(ctx, key, string(StatusDone), g.doneTTL).Err()
}

// Release: gọi khi xử lý THẤT BẠI → xóa key để cho phép retry
func (g *IdempotencyGuard) Release(ctx context.Context, messageID string) error {
	key := g.key(messageID)
	return g.rdb.Del(ctx, key).Err()
}
