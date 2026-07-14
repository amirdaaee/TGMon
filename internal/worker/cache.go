package worker

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/amirdaaee/TGMon/internal/log"
	"go.uber.org/zap"
)

// IFileCache defines a minimal disk-backed cache API with a convenience
// GetOrSet helper.
//
//go:generate mockgen -source=cache.go -destination=../../mocks/worker/cache.go -package=mocks_worker
type IFileCache[T any] interface {
	GetOrSet(string, func() (T, error)) (T, error)
}
type fileCache[T any] struct {
	root           string
	filenameSuffix string
	lock           sync.RWMutex
	ll             *zap.Logger
}

var _ IFileCache[any] = (*fileCache[any])(nil)

// getLocked returns the value stored for the key or an error if not present.
func (c *fileCache[T]) getLocked(key string) (T, error) {
	fp := c.getFilename(key)
	var zeroT T

	fi, err := os.Stat(fp)
	if err != nil {
		return zeroT, fmt.Errorf("error accessing cache file(%s): %w", fp, err)
	}
	if fi.IsDir() {
		return zeroT, fmt.Errorf("cache path is a directory: %s", fp)
	}

	data, err := os.ReadFile(fp)
	if err != nil {
		return zeroT, fmt.Errorf("error reading cache file(%s): %w", fp, err)
	}

	return c.unmarshallType(data)
}

// setLocked stores the provided value for the key, overwriting existing content.
func (c *fileCache[T]) setLocked(key string, value T) error {
	fp := c.getFilename(key)
	valMarshal, err := c.marshallType(value)
	if err != nil {
		return fmt.Errorf("error marshalling value: %w", err)
	}
	if err := os.WriteFile(fp, valMarshal, 0644); err != nil {
		return fmt.Errorf("error writing cache file(%s): %w", fp, err)
	}
	return nil
}

// GetOrSet tries to load the key, and on miss computes the value via fn,
// stores it, and returns it. Cache write failures are logged but not fatal.
func (c *fileCache[T]) GetOrSet(key string, fn func() (T, error)) (T, error) {
	ll := c.ll.Named("GetOrSet").With(zap.String("key", c.getCacheKey(key)))
	c.lock.RLock()
	// Try to get from cache first
	if value, err := c.getLocked(key); err == nil {
		c.lock.RUnlock()
		ll.Debug("cache hit")
		return value, nil
	}
	c.lock.RUnlock()
	c.lock.Lock()
	defer c.lock.Unlock()
	if value, err := c.getLocked(key); err == nil {
		return value, nil
	}
	ll.Debug("cache miss")

	// Cache miss: compute value
	value, err := fn()
	if err != nil {
		return value, err
	}

	// Store in cache (log error but don't fail)
	if err := c.setLocked(key, value); err != nil {
		ll.With(zap.Error(err)).Error("error setting cache")
	}

	return value, nil
}
func (c *fileCache[T]) unmarshallType(val []byte) (T, error) {
	var zeroT T

	switch v := any(zeroT).(type) {
	case int64:
		result, err := strconv.ParseInt(string(val), 10, 64)
		if err != nil {
			return zeroT, fmt.Errorf("failed to parse int64: %w", err)
		}
		return any(result).(T), nil
	case []byte:
		return any(val).(T), nil
	default:
		return zeroT, fmt.Errorf("unsupported type for cache: %T", v)
	}
}

func (c *fileCache[T]) marshallType(val T) ([]byte, error) {
	switch v := any(val).(type) {
	case int64:
		return []byte(strconv.FormatInt(v, 10)), nil
	case []byte:
		return v, nil
	default:
		return nil, fmt.Errorf("unsupported type for cache: %T", v)
	}
}
func (c *fileCache[T]) getCacheKey(key string) string {
	return fmt.Sprintf("%s-%s", key, c.filenameSuffix)
}
func (c *fileCache[T]) getFilename(key string) string {
	return filepath.Join(c.root, c.getCacheKey(key))
}

// NewAccessHashCache returns a cache specialized for int64 access hashes.
func NewAccessHashCache(root string) IFileCache[int64] {
	return &fileCache[int64]{root: root, filenameSuffix: "accHash", ll: log.Named(log.WorkerModule, "AccessHashCache")}
}

// NewDocCache returns a cache specialized for raw document bytes.
func NewDocCache(root string) IFileCache[[]byte] {
	return &fileCache[[]byte]{root: root, filenameSuffix: "doc", ll: log.Named(log.WorkerModule, "DocCache")}
}
