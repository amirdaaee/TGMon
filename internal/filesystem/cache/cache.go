package cache

import (
	"context"
	"fmt"
	"sync"
)

type DBCache[K comparable, V any] struct {
	cache       map[K]V
	lister      func(ctx context.Context) (map[K]V, error)
	isValid     bool
	isValidLock sync.RWMutex
	cacheLock   sync.RWMutex
}

func (dbc *DBCache[K, V]) Update(ctx context.Context) error {
	items, err := dbc.lister(ctx)
	if err != nil {
		return err
	}
	dbc.cacheLock.Lock()
	defer dbc.cacheLock.Unlock()
	dbc.cache = items
	return nil
}

func (dbc *DBCache[K, V]) List(ctx context.Context) ([]V, error) {
	if err := dbc.mayUpdate(ctx); err != nil {
		return nil, err
	}
	dbc.cacheLock.RLock()
	defer dbc.cacheLock.RUnlock()
	items := make([]V, 0, len(dbc.cache))
	for _, item := range dbc.cache {
		items = append(items, item)
	}
	return items, nil
}
func (dbc *DBCache[K, V]) Find(ctx context.Context, key K) (V, error) {
	var v V
	if err := dbc.mayUpdate(ctx); err != nil {
		return v, err
	}
	dbc.cacheLock.RLock()
	defer dbc.cacheLock.RUnlock()
	v, ok := dbc.cache[key]
	if !ok {
		return v, CacheNotFoundError
	}
	return v, nil
}
func (dbc *DBCache[K, V]) Exists(ctx context.Context, key K) (bool, error) {
	if err := dbc.mayUpdate(ctx); err != nil {
		return false, err
	}
	dbc.cacheLock.RLock()
	defer dbc.cacheLock.RUnlock()
	_, ok := dbc.cache[key]
	return ok, nil
}
func (dbc *DBCache[K, V]) Invalidate(ctx context.Context) {
	dbc.isValidLock.Lock()
	defer dbc.isValidLock.Unlock()
	dbc.isValid = false
}
func (dbc *DBCache[K, V]) mayUpdate(ctx context.Context) error {
	dbc.isValidLock.RLock()
	if dbc.isValid {
		dbc.isValidLock.RUnlock()
		return nil
	}
	dbc.isValidLock.RUnlock()

	dbc.isValidLock.Lock()
	defer dbc.isValidLock.Unlock()
	if dbc.isValid {
		return nil
	}
	if err := dbc.Update(ctx); err != nil {
		return fmt.Errorf("failed to update cache: %w", err)
	}
	dbc.isValid = true
	return nil
}

func NewDBCacher[K comparable, V any](lister func(ctx context.Context) (map[K]V, error)) *DBCache[K, V] {
	return &DBCache[K, V]{
		cache:       make(map[K]V),
		lister:      lister,
		isValid:     false,
		isValidLock: sync.RWMutex{},
		cacheLock:   sync.RWMutex{},
	}
}
