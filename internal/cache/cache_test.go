package cache

import (
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
