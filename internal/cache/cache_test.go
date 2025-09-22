package cache

import (
	"bytes"
	"strings"
	"testing"
)

func TestPutUpdatesExistingEntryWithoutGrowingSize(t *testing.T) {
	cache, err := New(1)
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}

	key1 := "alpha"
	key2 := "beta"
	initialValue := strings.Repeat("x", 16)
	updatedValue := strings.Repeat("y", 24)

	if err := cache.Put(key1, initialValue); err != nil {
		t.Fatalf("put initial key1 failed: %v", err)
	}

	if err := cache.Put(key2, "value"); err != nil {
		t.Fatalf("put key2 failed: %v", err)
	}

	sizeBeforeUpdate := cache.SizeOf()
	key1OriginalSize := sizeof(&Entry{Key: key1, Value: initialValue})
	key1UpdatedSize := sizeof(&Entry{Key: key1, Value: updatedValue})

	if err := cache.Put(key1, updatedValue); err != nil {
		t.Fatalf("put updated key1 failed: %v", err)
	}

	expectedSize := sizeBeforeUpdate - int64(key1OriginalSize) + int64(key1UpdatedSize)
	if cache.SizeOf() != expectedSize {
		t.Fatalf("unexpected cache size: got %d, want %d", cache.SizeOf(), expectedSize)
	}

	if cache.SizeOf() > 1*1024*1024 {
		t.Fatalf("cache size exceeded limit: %d", cache.SizeOf())
	}

	if value, hit, err := cache.Get(key2); err != nil || !hit || value != "value" {
		t.Fatalf("expected key2 to remain in cache, hit=%v err=%v value=%v", hit, err, value)
	}
}

func TestPutHandlesNonStringKeyAndValue(t *testing.T) {
	cache, err := New(1)
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}

	key := 123
	value := []byte{1, 2, 3, 4, 5}

	if err := cache.Put(key, value); err != nil {
		t.Fatalf("put non-string key/value failed: %v", err)
	}

	expectedSize := sizeof(&Entry{Key: key, Value: value})
	if cache.SizeOf() != int64(expectedSize) {
		t.Fatalf("unexpected cache size: got %d, want %d", cache.SizeOf(), expectedSize)
	}
}

type customKey struct {
	name string
}

func TestEvictionWithNonStringKeyAndValue(t *testing.T) {
	cache, err := New(1)
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}

	key1 := customKey{name: "first"}
	value1 := bytes.Repeat([]byte("a"), 700*1024)
	if err := cache.Put(key1, value1); err != nil {
		t.Fatalf("put key1 failed: %v", err)
	}

	key2 := customKey{name: "second"}
	value2 := bytes.Repeat([]byte("b"), 700*1024)
	if err := cache.Put(key2, value2); err != nil {
		t.Fatalf("put key2 failed: %v", err)
	}

	if _, hit, err := cache.Get(key1); err != nil || hit {
		t.Fatalf("expected key1 to be evicted, hit=%v err=%v", hit, err)
	}

	expectedSize := sizeof(&Entry{Key: key2, Value: value2})
	if cache.SizeOf() != int64(expectedSize) {
		t.Fatalf("unexpected cache size after eviction: got %d, want %d", cache.SizeOf(), expectedSize)
	}
}
