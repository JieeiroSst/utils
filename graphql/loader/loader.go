package loader

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type LoaderConfig struct {
	Wait time.Duration
	MaxBatch int
}

var DefaultLoaderConfig = LoaderConfig{
	Wait:     5 * time.Millisecond,
	MaxBatch: 100,
}


type BatchFunc func(ctx context.Context, keys []string) (map[string]interface{}, error)

type Loader struct {
	fetch     BatchFunc
	config    LoaderConfig
	batch     *batch
	mu        sync.Mutex
	cacheMap  map[string]*result
	cacheMu   sync.RWMutex
	useCache  bool
}

type result struct {
	data interface{}
	err  error
}

type batch struct {
	keys    []string
	data    map[string]*result
	done    chan struct{}
	closing bool
	mu      sync.Mutex
}

func NewLoader(fetch BatchFunc) *Loader {
	return NewLoaderWithConfig(fetch, DefaultLoaderConfig)
}

func NewLoaderWithConfig(fetch BatchFunc, config LoaderConfig) *Loader {
	return &Loader{
		fetch:    fetch,
		config:   config,
		cacheMap: make(map[string]*result),
		useCache: true,
	}
}

func (l *Loader) DisableCache() *Loader {
	l.useCache = false
	return l
}

func (l *Loader) ClearCache() {
	l.cacheMu.Lock()
	defer l.cacheMu.Unlock()
	l.cacheMap = make(map[string]*result)
}

func (l *Loader) ClearKey(key string) {
	l.cacheMu.Lock()
	defer l.cacheMu.Unlock()
	delete(l.cacheMap, key)
}

func (l *Loader) Load(ctx context.Context, key string) (interface{}, error) {
	if l.useCache {
		l.cacheMu.RLock()
		if res, ok := l.cacheMap[key]; ok {
			l.cacheMu.RUnlock()
			return res.data, res.err
		}
		l.cacheMu.RUnlock()
	}
	
	l.mu.Lock()
	
	if l.batch == nil {
		l.batch = &batch{
			keys: []string{},
			data: make(map[string]*result),
			done: make(chan struct{}),
		}
		
		go l.executeBatch(ctx)
	}
	
	b := l.batch
	b.keys = append(b.keys, key)
	
	if len(b.keys) >= l.config.MaxBatch {
		go l.executeBatch(ctx)
	}
	
	l.mu.Unlock()
	
	<-b.done
	
	res := b.data[key]
	if res == nil {
		return nil, fmt.Errorf("key %s not found in batch result", key)
	}
	
	if l.useCache && res.err == nil {
		l.cacheMu.Lock()
		l.cacheMap[key] = res
		l.cacheMu.Unlock()
	}
	
	return res.data, res.err
}

func (l *Loader) LoadMany(ctx context.Context, keys []string) ([]interface{}, []error) {
	results := make([]interface{}, len(keys))
	errs := make([]error, len(keys))
	
	var wg sync.WaitGroup
	wg.Add(len(keys))
	
	for i, key := range keys {
		go func(idx int, k string) {
			defer wg.Done()
			data, err := l.Load(ctx, k)
			results[idx] = data
			errs[idx] = err
		}(i, key)
	}
	
	wg.Wait()
	return results, errs
}

func (l *Loader) Prime(key string, data interface{}) {
	if !l.useCache {
		return
	}
	
	l.cacheMu.Lock()
	defer l.cacheMu.Unlock()
	
	l.cacheMap[key] = &result{
		data: data,
		err:  nil,
	}
}

func (l *Loader) executeBatch(ctx context.Context) {
	time.Sleep(l.config.Wait)
	
	l.mu.Lock()
	if l.batch == nil || l.batch.closing {
		l.mu.Unlock()
		return
	}
	
	b := l.batch
	b.closing = true
	l.batch = nil 
	l.mu.Unlock()
	
	keys := b.keys
	results, err := l.fetch(ctx, keys)
	
	b.mu.Lock()
	defer b.mu.Unlock()
	
	if err != nil {
		for _, key := range keys {
			b.data[key] = &result{
				data: nil,
				err:  err,
			}
		}
	} else {
		for _, key := range keys {
			if data, ok := results[key]; ok {
				b.data[key] = &result{
					data: data,
					err:  nil,
				}
			} else {
				b.data[key] = &result{
					data: nil,
					err:  fmt.Errorf("key %s not found", key),
				}
			}
		}
	}
	
	close(b.done)
}

type LoaderMap struct {
	loaders map[string]*Loader
	mu      sync.RWMutex
}

func NewLoaderMap() *LoaderMap {
	return &LoaderMap{
		loaders: make(map[string]*Loader),
	}
}

func (lm *LoaderMap) Register(entityType string, loader *Loader) {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	lm.loaders[entityType] = loader
}

func (lm *LoaderMap) Get(entityType string) (*Loader, bool) {
	lm.mu.RLock()
	defer lm.mu.RUnlock()
	loader, ok := lm.loaders[entityType]
	return loader, ok
}

func (lm *LoaderMap) ClearAll() {
	lm.mu.RLock()
	defer lm.mu.RUnlock()
	
	for _, loader := range lm.loaders {
		loader.ClearCache()
	}
}
