package cache

import (
	"container/list"
)

type LRUCache struct {
	size      int
	evictList *list.List
	items     map[interface{}]*list.Element
}

type entry struct {
	key   interface{}
	value interface{}
}

func NewLRUCache(size int) *LRUCache {
	return &LRUCache{
		size:      size,
		evictList: list.New(),
		items:     make(map[interface{}]*list.Element),
	}
}

func (c *LRUCache) Get(key interface{}) (value interface{}, ok bool) {
	if ele, hit := c.items[key]; hit {
		c.evictList.MoveToFront(ele)
		return ele.Value.(*entry).value, true
	}
	return
}

func (c *LRUCache) Put(key, value interface{}) {
	if ele, hit := c.items[key]; hit {
		c.evictList.MoveToFront(ele)
		ele.Value.(*entry).value = value
		return
	}

	ele := c.evictList.PushFront(&entry{key, value})
	c.items[key] = ele

	if c.evictList.Len() > c.size {
		c.removeOldest()
	}
}

func (c *LRUCache) removeOldest() {
	ele := c.evictList.Back()
	if ele != nil {
		c.removeElement(ele)
	}
}

func (c *LRUCache) removeElement(e *list.Element) {
	c.evictList.Remove(e)
	kv := e.Value.(*entry)
	delete(c.items, kv.key)
}
