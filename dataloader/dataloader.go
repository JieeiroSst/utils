package dataloader

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
	"gorm.io/gorm"
)

type DBType string

const (
	MySQL      DBType = "mysql"
	PostgreSQL DBType = "postgresql"
)

type DataLoader struct {
	maxBatchSize int
	wait         time.Duration
	dbType       DBType
	cacheTTL     time.Duration
	mu           sync.Mutex
	cache        map[string]cacheItem
}

type cacheItem struct {
	data      interface{}
	expiresAt time.Time
}

type DataLoaderOption func(*DataLoader)

func WithMaxBatchSize(size int) DataLoaderOption {
	return func(d *DataLoader) {
		d.maxBatchSize = size
	}
}

func WithWait(wait time.Duration) DataLoaderOption {
	return func(d *DataLoader) {
		d.wait = wait
	}
}

func WithCacheTTL(ttl time.Duration) DataLoaderOption {
	return func(d *DataLoader) {
		d.cacheTTL = ttl
	}
}

func WithDBType(dbType DBType) DataLoaderOption {
	return func(d *DataLoader) {
		d.dbType = dbType
	}
}

func NewDataLoader(opts ...DataLoaderOption) *DataLoader {
	dl := &DataLoader{
		maxBatchSize: 100,
		wait:         time.Millisecond * 5,
		dbType:       MySQL,
		cacheTTL:     time.Minute * 5,
		cache:        make(map[string]cacheItem),
	}

	for _, opt := range opts {
		opt(dl)
	}

	return dl
}

type SQLLoader struct {
	*DataLoader
	db *sqlx.DB
}

func NewSQLLoader(db *sql.DB, opts ...DataLoaderOption) *SQLLoader {
	return &SQLLoader{
		DataLoader: NewDataLoader(opts...),
		db:         sqlx.NewDb(db, ""),
	}
}

func (l *SQLLoader) LoadMany(ctx context.Context, table string, column string, keys []interface{}, dest interface{}) error {
	if len(keys) == 0 {
		return nil
	}

	cacheKey := fmt.Sprintf("%s:%s:%v", table, column, keys)

	l.mu.Lock()
	if item, ok := l.cache[cacheKey]; ok && time.Now().Before(item.expiresAt) {
		l.mu.Unlock()
		destVal := reflect.ValueOf(dest).Elem()
		srcVal := reflect.ValueOf(item.data)
		destVal.Set(srcVal)
		return nil
	}
	l.mu.Unlock()

	placeholder := "?"
	if l.dbType == PostgreSQL {
		placeholder = "$%d"
	}

	placeholders := make([]string, len(keys))
	args := make([]interface{}, len(keys))
	for i, key := range keys {
		if l.dbType == PostgreSQL {
			placeholders[i] = fmt.Sprintf(placeholder, i+1)
		} else {
			placeholders[i] = placeholder
		}
		args[i] = key
	}

	query := fmt.Sprintf("SELECT * FROM %s WHERE %s IN (%s)",
		table, column, strings.Join(placeholders, ","))

	err := l.db.SelectContext(ctx, dest, query, args...)
	if err != nil {
		return err
	}

	l.mu.Lock()
	l.cache[cacheKey] = cacheItem{
		data:      reflect.ValueOf(dest).Elem().Interface(),
		expiresAt: time.Now().Add(l.cacheTTL),
	}
	l.mu.Unlock()

	return nil
}

func (l *SQLLoader) LoadOne(ctx context.Context, table string, column string, key interface{}, dest interface{}) error {
	return l.LoadMany(ctx, table, column, []interface{}{key}, dest)
}

type GORMLoader struct {
	*DataLoader
	db *gorm.DB
}

func NewGORMLoader(db *gorm.DB, opts ...DataLoaderOption) *GORMLoader {
	return &GORMLoader{
		DataLoader: NewDataLoader(opts...),
		db:         db,
	}
}

func (l *GORMLoader) LoadMany(ctx context.Context, model interface{}, column string, keys []interface{}, dest interface{}) error {
	if len(keys) == 0 {
		return nil
	}

	modelType := reflect.TypeOf(model).String()
	cacheKey := fmt.Sprintf("%s:%s:%v", modelType, column, keys)

	l.mu.Lock()
	if item, ok := l.cache[cacheKey]; ok && time.Now().Before(item.expiresAt) {
		l.mu.Unlock()
		destVal := reflect.ValueOf(dest).Elem()
		srcVal := reflect.ValueOf(item.data)
		destVal.Set(srcVal)
		return nil
	}
	l.mu.Unlock()

	tx := l.db.WithContext(ctx).Model(model).Where(fmt.Sprintf("%s IN ?", column), keys).Find(dest)
	if tx.Error != nil {
		return tx.Error
	}

	l.mu.Lock()
	l.cache[cacheKey] = cacheItem{
		data:      reflect.ValueOf(dest).Elem().Interface(),
		expiresAt: time.Now().Add(l.cacheTTL),
	}
	l.mu.Unlock()

	return nil
}

func (l *GORMLoader) LoadOne(ctx context.Context, model interface{}, column string, key interface{}, dest interface{}) error {
	return l.LoadMany(ctx, model, column, []interface{}{key}, dest)
}

type BatchLoader struct {
	*DataLoader
	batchFn    func(ctx context.Context, keys []interface{}) (map[interface{}]interface{}, error)
	batchChan  chan *batchRequest
	closeChan  chan struct{}
	batchMutex sync.Mutex
	batch      *batch
}

type batch struct {
	keys     []interface{}
	requests []*batchRequest
}

type batchRequest struct {
	key      interface{}
	resultCh chan batchResult
}

type batchResult struct {
	data interface{}
	err  error
}

func NewBatchLoader(batchFn func(ctx context.Context, keys []interface{}) (map[interface{}]interface{}, error), opts ...DataLoaderOption) *BatchLoader {
	dl := NewDataLoader(opts...)

	l := &BatchLoader{
		DataLoader: dl,
		batchFn:    batchFn,
		batchChan:  make(chan *batchRequest),
		closeChan:  make(chan struct{}),
	}

	go l.batchProcessor()

	return l
}

func (l *BatchLoader) batchProcessor() {
	for {
		select {
		case req := <-l.batchChan:
			l.batchMutex.Lock()
			if l.batch == nil {
				l.batch = &batch{
					keys:     make([]interface{}, 0, l.maxBatchSize),
					requests: make([]*batchRequest, 0, l.maxBatchSize),
				}
				time.AfterFunc(l.wait, l.dispatchBatch)
			}
			l.batch.keys = append(l.batch.keys, req.key)
			l.batch.requests = append(l.batch.requests, req)
			l.batchMutex.Unlock()

		case <-l.closeChan:
			return
		}
	}
}

func (l *BatchLoader) dispatchBatch() {
	l.batchMutex.Lock()
	currentBatch := l.batch
	l.batch = nil
	l.batchMutex.Unlock()

	if currentBatch == nil {
		return
	}

	ctx := context.Background()
	results, err := l.batchFn(ctx, currentBatch.keys)

	if err != nil {
		for _, req := range currentBatch.requests {
			req.resultCh <- batchResult{err: err}
		}
		return
	}

	for _, req := range currentBatch.requests {
		data, ok := results[req.key]
		if !ok {
			req.resultCh <- batchResult{err: fmt.Errorf("no result for key: %v", req.key)}
			continue
		}
		req.resultCh <- batchResult{data: data}
	}
}

func (l *BatchLoader) Load(ctx context.Context, key interface{}) (interface{}, error) {
	cacheKey := fmt.Sprintf("%v", key)

	l.mu.Lock()
	if item, ok := l.cache[cacheKey]; ok && time.Now().Before(item.expiresAt) {
		l.mu.Unlock()
		return item.data, nil
	}
	l.mu.Unlock()

	resultCh := make(chan batchResult, 1)
	req := &batchRequest{
		key:      key,
		resultCh: resultCh,
	}

	select {
	case l.batchChan <- req:
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	var result batchResult
	select {
	case result = <-resultCh:
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	if result.err == nil {
		l.mu.Lock()
		l.cache[cacheKey] = cacheItem{
			data:      result.data,
			expiresAt: time.Now().Add(l.cacheTTL),
		}
		l.mu.Unlock()
	}

	return result.data, result.err
}

func (l *BatchLoader) Close() {
	close(l.closeChan)
}
