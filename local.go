package easycache

import (
	"context"
	"errors"
	"sync"
	"time"
)

var errKeyNotFound = errors.New("[easy-cache] key not found")

var _ Cache = (*LocalCache)(nil)

type LocalCache struct {
	mu   sync.RWMutex
	data map[string]cacheItem

	interval time.Duration
	close    chan struct{}
}

func (lc *LocalCache) Set(_ context.Context, key string, val any, expires time.Duration) error {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	var expireAt time.Time
	if expires > 0 {
		expireAt = time.Now().Add(expires)
	}

	lc.data[key] = cacheItem{
		val:      val,
		expireAt: expireAt,
	}
	return nil
}

func (lc *LocalCache) Get(_ context.Context, key string) (any, error) {
	lc.mu.RLock()
	item, ok := lc.data[key]
	lc.mu.RUnlock()

	if !ok {
		return nil, errKeyNotFound
	}

	now := time.Now()
	if item.Expired(now) {
		// key 已过期
		lc.mu.Lock()
		defer lc.mu.Unlock()

		// double check
		// 防止期间用户重新设置 key 刷新了过期时间
		item, ok = lc.data[key]
		if ok && item.Expired(now) {
			delete(lc.data, key)
		}

		return nil, errKeyNotFound
	}
	return item.val, nil
}

func (lc *LocalCache) Del(_ context.Context, key string) error {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	delete(lc.data, key)
	return nil
}

func (lc *LocalCache) Close() error {
	close(lc.close)
	return nil
}

func (lc *LocalCache) cleanExpiredKey() {
	ticker := time.NewTicker(lc.interval)

	for {
		select {
		case now := <-ticker.C:
			lc.mu.Lock()

			count := 0
			for key, item := range lc.data {
				if count >= 1024 {
					break
				}
				count++
				if item.Expired(now) {
					delete(lc.data, key)
				}
			}
			lc.mu.Unlock()
		case <-lc.close:
			return
		}
	}
}

type LocalCacheBuilder struct {
	interval time.Duration
}

func (lcb *LocalCacheBuilder) WithInterval(interval time.Duration) *LocalCacheBuilder {
	lcb.interval = interval
	return lcb
}

func (lcb *LocalCacheBuilder) Build() *LocalCache {

	lc := &LocalCache{
		data:     make(map[string]cacheItem),
		interval: lcb.interval,
		close:    make(chan struct{}),
	}

	go lc.cleanExpiredKey()

	return lc
}

func NewLocalCacheBuilder() *LocalCacheBuilder {
	return &LocalCacheBuilder{
		interval: 15 * time.Second,
	}
}

type cacheItem struct {
	val      any
	expireAt time.Time
}

func (ci *cacheItem) Expired(now time.Time) bool {
	if !ci.expireAt.IsZero() && now.After(ci.expireAt) {
		return true
	}
	return false
}
