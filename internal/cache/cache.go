package cache

import (
	"container/list"
	"errors"
	"log"
	"sync"
)

// Cache represents a least recently used (LRU) cache.
type Cache struct {
	items        map[interface{}]*list.Element
	evictionList *list.List
	mutex        sync.Mutex
	size         int
}

// Entry represents a key-value pair in the cache.
type Entry struct {
	Key   interface{}
	Value interface{}
}

// New creates a new LRU cache with the specified size.
func New(size int) (*Cache, error) {
	if size <= 0 {
		return nil, errors.New("cache size must be positive")
	}
	return &Cache{
		size:         size,
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

	if ele, hit := c.items[key]; hit {
		c.evictionList.MoveToFront(ele)
		ele.Value.(*Entry).Value = value
		return nil
	}

	ele := c.evictionList.PushFront(&Entry{Key: key, Value: value})
	c.items[key] = ele

	if c.evictionList.Len() > c.size {
		c.removeOldest()
	}
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
}
