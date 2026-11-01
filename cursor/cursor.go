package cursor

import (
	"cmp"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
)

type sliceSource[T any, K cmp.Ordered] struct {
	items []T
	key   func(T) K
}

func NewSliceSource[T any, K cmp.Ordered](items []T, key func(T) K) Source[T, K] {
	return &sliceSource[T, K]{items: items, key: key}
}

func (s *sliceSource[T, K]) Fetch(_ context.Context, req Request[K]) (Result[T, K], error) {
	start := 0
	if req.After != nil {
		// vị trí đầu tiên có key > after
		start = sort.Search(len(s.items), func(i int) bool {
			return s.key(s.items[i]) > *req.After
		})
	}
	end := start + req.Limit
	hasMore := end < len(s.items)
	if end > len(s.items) {
		end = len(s.items)
	}
	res := Result[T, K]{Items: s.items[start:end], HasMore: hasMore}
	if hasMore {
		k := s.key(res.Items[len(res.Items)-1])
		res.Next = &k
	}
	return res, nil
}

type SQLSource[T, K any] struct {
	DB         *sql.DB
	FirstQuery string
	NextQuery  string
	ArgsOf     func(K) []any
	Scan       func(*sql.Rows) (T, error)
	KeyOf      func(T) K
}

func (s *SQLSource[T, K]) Fetch(ctx context.Context, req Request[K]) (Result[T, K], error) {
	var (
		rows *sql.Rows
		err  error
		lim  = req.Limit + 1
	)
	if req.After == nil {
		rows, err = s.DB.QueryContext(ctx, s.FirstQuery, lim)
	} else {
		args := append(s.ArgsOf(*req.After), lim)
		rows, err = s.DB.QueryContext(ctx, s.NextQuery, args...)
	}
	if err != nil {
		return Result[T, K]{}, err
	}
	defer rows.Close()

	items := make([]T, 0, lim)
	for rows.Next() {
		it, err := s.Scan(rows)
		if err != nil {
			return Result[T, K]{}, err
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return Result[T, K]{}, err
	}

	res := Result[T, K]{}
	if len(items) > req.Limit {
		items = items[:req.Limit]
		res.HasMore = true
	}
	res.Items = items
	if res.HasMore {
		k := s.KeyOf(items[len(items)-1])
		res.Next = &k
	}
	return res, nil
}

func NewOffsetSource[T any](
	fetch func(ctx context.Context, offset, limit int) ([]T, error),
) Source[T, int] {
	return SourceFunc[T, int](func(ctx context.Context, req Request[int]) (Result[T, int], error) {
		offset := 0
		if req.After != nil && *req.After > 0 {
			offset = *req.After
		}
		items, err := fetch(ctx, offset, req.Limit+1)
		if err != nil {
			return Result[T, int]{}, err
		}
		res := Result[T, int]{}
		if len(items) > req.Limit {
			items = items[:req.Limit]
			res.HasMore = true
		}
		res.Items = items
		if res.HasMore {
			next := offset + len(items)
			res.Next = &next
		}
		return res, nil
	})
}

func NewTokenSource[T any](
	fetch func(ctx context.Context, token string, limit int) (items []T, nextToken string, err error),
) Source[T, string] {
	return SourceFunc[T, string](func(ctx context.Context, req Request[string]) (Result[T, string], error) {
		token := ""
		if req.After != nil {
			token = *req.After
		}
		items, next, err := fetch(ctx, token, req.Limit)
		if err != nil {
			return Result[T, string]{}, err
		}
		res := Result[T, string]{Items: items}
		if next != "" {
			res.HasMore = true
			res.Next = &next
		}
		return res, nil
	})
}

type Request[K any] struct {
	After *K
	Limit int
}

type Result[T, K any] struct {
	Items   []T
	Next    *K
	HasMore bool
}

type Source[T, K any] interface {
	Fetch(ctx context.Context, req Request[K]) (Result[T, K], error)
}

type SourceFunc[T, K any] func(ctx context.Context, req Request[K]) (Result[T, K], error)

func (f SourceFunc[T, K]) Fetch(ctx context.Context, req Request[K]) (Result[T, K], error) {
	return f(ctx, req)
}

type Page[T any] struct {
	Data       []T    `json:"data"`
	NextCursor string `json:"next_cursor,omitempty"`
	HasNext    bool   `json:"has_next"`
}

type Paginator[T, K any] struct {
	src          Source[T, K]
	codec        *Codec
	DefaultLimit int
	MaxLimit     int
}

func New[T, K any](src Source[T, K], codec *Codec) *Paginator[T, K] {
	if codec == nil {
		codec = NewCodec(nil)
	}
	return &Paginator[T, K]{src: src, codec: codec, DefaultLimit: 20, MaxLimit: 100}
}

func (p *Paginator[T, K]) Page(ctx context.Context, cursor string, limit int) (*Page[T], error) {
	if limit <= 0 {
		limit = p.DefaultLimit
	}
	if limit > p.MaxLimit {
		limit = p.MaxLimit
	}

	req := Request[K]{Limit: limit}
	if cursor != "" {
		var k K
		if err := p.codec.Decode(cursor, &k); err != nil {
			return nil, err
		}
		req.After = &k
	}

	res, err := p.src.Fetch(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("cursorpage: fetch: %w", err)
	}

	page := &Page[T]{Data: res.Items, HasNext: res.HasMore}
	if page.Data == nil {
		page.Data = []T{} // JSON trả [] thay vì null
	}
	if res.HasMore && res.Next != nil {
		tok, err := p.codec.Encode(res.Next)
		if err != nil {
			return nil, err
		}
		page.NextCursor = tok
	} else {
		page.HasNext = false
	}
	return page, nil
}

var ErrInvalidCursor = errors.New("cursorpage: invalid cursor")

type Codec struct {
	secret []byte
}

func NewCodec(secret []byte) *Codec { return &Codec{secret: secret} }

func (c *Codec) Encode(v any) (string, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	payload := base64.RawURLEncoding.EncodeToString(raw)
	if len(c.secret) == 0 {
		return payload, nil
	}
	return payload + "." + c.sign(payload), nil
}

func (c *Codec) Decode(token string, out any) error {
	payload := token
	if len(c.secret) > 0 {
		i := strings.LastIndexByte(token, '.')
		if i < 0 {
			return ErrInvalidCursor
		}
		sig := token[i+1:]
		payload = token[:i]
		if !hmac.Equal([]byte(sig), []byte(c.sign(payload))) {
			return ErrInvalidCursor
		}
	}
	raw, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		return ErrInvalidCursor
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return ErrInvalidCursor
	}
	return nil
}

func (c *Codec) sign(payload string) string {
	m := hmac.New(sha256.New, c.secret)
	m.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(m.Sum(nil))
}
