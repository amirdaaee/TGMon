package stream

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/amirdaaee/TGMon/internal/log"
	"github.com/sirupsen/logrus"
)

//go:generate mockgen -source=cache.go -destination=../../mocks/stream/cache.go -package=mocks
type IFileCache[T any] interface {
	Get(string) (T, error)
	Set(string, T) error
	GetOrSet(string, func() (T, error)) (T, error)
}
type fileCache[T any] struct {
	root           string
	filenameSuffix string
}

var _ IFileCache[any] = (*fileCache[any])(nil)

func (c *fileCache[T]) Get(key string) (T, error) {
	fp := c.getFilename(key)
	zeroT := new(T)
	if fi, err := os.Stat(fp); err == nil && !fi.IsDir() {
		if data, err := os.ReadFile(fp); err == nil {
			return c.unmarshallType(data)
		} else {
			return *zeroT, fmt.Errorf("error reading cache file(%s): %w", fp, err)
		}
	} else {
		return *zeroT, fmt.Errorf("error accessing cache file(%s): %w", fp, err)
	}
}
func (c *fileCache[T]) Set(key string, value T) error {
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
func (c *fileCache[T]) GetOrSet(key string, fn func() (T, error)) (T, error) {
	ll := c.getLogger("GetOrSet").WithField("key", c.getCacheKey(key))
	var value T
	if v, err := c.Get(key); err == nil {
		ll.Debug("cache hit")
		return v, nil
	} else {
		ll.Debugf("cache miss: (%s)", err)
		v, err := fn()
		if err != nil {
			return v, err
		}
		value = v
	}
	if err := c.Set(key, value); err != nil {
		ll.WithError(err).Error("error setting cache")
	}
	return value, nil
}
func (c *fileCache[T]) unmarshallType(val []byte) (T, error) {
	var result T
	var err error
	var t T
	switch any(t).(type) {
	case int64:
		var v int64
		v, err = strconv.ParseInt(string(val), 10, 64)
		result = any(v).(T)
	case []byte:
		result = any(val).(T)
	default:
		err = fmt.Errorf("unsupported type for cache: %T", t)
	}
	if err != nil {
		return t, err
	}
	return result, nil
}
func (c *fileCache[T]) marshallType(val T) ([]byte, error) {
	var t T
	switch any(t).(type) {
	case int64:
		strVal := strconv.FormatInt(any(val).(int64), 10)
		return []byte(strVal), nil
	case []byte:
		return any(val).([]byte), nil
	default:
		return nil, fmt.Errorf("unsupported type for cache: %T", t)
	}
}
func (c *fileCache[T]) getCacheKey(key string) string {
	return fmt.Sprintf("%s-%s", key, c.filenameSuffix)
}
func (c *fileCache[T]) getFilename(key string) string {
	return filepath.Join(c.root, c.getCacheKey(key))
}
func (c *fileCache[T]) getLogger(fn string) *logrus.Entry {
	return log.GetLogger(log.StreamModule).WithField("func", fmt.Sprintf("%T.%s", c, fn))
}
func NewAccessHashCache(root string) IFileCache[int64] {
	return &fileCache[int64]{root: root, filenameSuffix: "accHash"}
}
func NewDocCache(root string) IFileCache[[]byte] {
	return &fileCache[[]byte]{root: root, filenameSuffix: "doc"}
}
