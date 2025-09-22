package cache

import (
	"container/list"
	"errors"
	"fmt"
	"log"
	"sync"
	"unsafe"
)

const absurdSizeLimit = 5 * 1024 * 1024

// Cache represents a least recently used (LRU) cache.
type Cache struct {
	items        map[interface{}]*list.Element
	evictionList *list.List
	mutex        sync.Mutex
	currentSize  int64
	maxSizeBytes int64
}

// Entry represents a key-value pair in the cache.
type Entry struct {
	Key   interface{}
	Value interface{}
}

// New creates a new LRU cache with the specified size.
func New(maxSizeMB int64) (*Cache, error) {
	if maxSizeMB <= 0 {
		return nil, errors.New("cache max size must be positive")
	}
	return &Cache{
		maxSizeBytes: maxSizeMB * 1024 * 1024,
		evictionList: list.New(),
		items:        make(map[interface{}]*list.Element),
	}, nil
}

// Get retrieves the value associated with the given key.
func (c *Cache) Get(key interface{}) (interface{}, bool, error) {
	if key == nil {
		return nil, false, errors.New("key cannot be nil")
	}

	defer func() {
		if r := recover(); r != nil {
			log.Printf("recovered from panic in Get: %v", r)
		}
	}()

	c.mutex.Lock()
	defer c.mutex.Unlock()

	if ele, hit := c.items[key]; hit {
		c.evictionList.MoveToFront(ele)
		return ele.Value.(*Entry).Value, true, nil
	}
	return nil, false, nil
}

// Put adds or updates a key-value pair in the cache.
func (c *Cache) Put(key, value interface{}) error {
	if key == nil {
		return errors.New("key cannot be nil")
	}
	if value == nil {
		return errors.New("value cannot be nil")
	}

	defer func() {
		if r := recover(); r != nil {
			log.Printf("recovered from panic in Put: %v", r)
		}
	}()

	c.mutex.Lock()
	defer c.mutex.Unlock()

	itemSize := sizeof(&Entry{Key: key, Value: value})
	if itemSize > absurdSizeLimit {
		return errors.New("item size exceeds absurd size limit")
	}

	if ele, hit := c.items[key]; hit {
		entry := ele.Value.(*Entry)
		previousSize := sizeof(entry)

		c.evictionList.Remove(ele)
		delete(c.items, key)
		c.currentSize -= int64(previousSize)

		entry.Value = value
		updatedSize := sizeof(entry)

		for c.currentSize+int64(updatedSize) > c.maxSizeBytes {
			c.removeOldest()
		}

		ele = c.evictionList.PushFront(entry)
		c.items[key] = ele
		c.currentSize += int64(updatedSize)
		return nil
	}

	for c.currentSize+int64(itemSize) > c.maxSizeBytes {
		c.removeOldest()
	}

	ele := c.evictionList.PushFront(&Entry{Key: key, Value: value})
	c.items[key] = ele
	c.currentSize += int64(itemSize)
	return nil
}

// removeOldest removes the least recently used item from the cache.
func (c *Cache) removeOldest() {
	ele := c.evictionList.Back()
	if ele != nil {
		c.removeElement(ele)
	}
}

// removeElement removes an element from the cache.
func (c *Cache) removeElement(e *list.Element) {
	c.evictionList.Remove(e)
	kv := e.Value.(*Entry)
	delete(c.items, kv.Key)
	c.currentSize -= int64(sizeof(kv))
}

// SizeOf returns the approximate memory usage in bytes of the cache.
func (c *Cache) SizeOf() int64 {
	return c.currentSize
}

// sizeof returns the approximate size of the Entry object in bytes.
func sizeof(e *Entry) int {
	size := int(unsafe.Sizeof(*e))
	size += len(e.Key.(string))
	size += len(e.Value.(string))
	return size
}

type ByteSize float64

const (
	_           = iota
	KB ByteSize = 1 << (10 * iota)
	MB
	GB
	TB
	PB
	EB
	ZB
	YB
)

// String returns a human-readable representation of a ByteSize.
func (b ByteSize) String() string {
	switch {
	case b >= YB:
		return fmt.Sprintf("%.2fYB", b/YB)
	case b >= ZB:
		return fmt.Sprintf("%.2fZB", b/ZB)
	case b >= EB:
		return fmt.Sprintf("%.2fEB", b/EB)
	case b >= PB:
		return fmt.Sprintf("%.2fPB", b/PB)
	case b >= TB:
		return fmt.Sprintf("%.2fTB", b/TB)
	case b >= GB:
		return fmt.Sprintf("%.2fGB", b/GB)
	case b >= MB:
		return fmt.Sprintf("%.2fMB", b/MB)
	case b >= KB:
		return fmt.Sprintf("%.2fKB", b/KB)
	}
	return fmt.Sprintf("%.2fB", b)
}

// returns a human-readable size format for the given number of bytes.
func ReadableSize(bytes int64) string {
	return ByteSize(bytes).String()
}
