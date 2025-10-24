package rv

import (
	"errors"
	"testing"
	"time"
)

func TestLockUnlock(t *testing.T) {
	v := NewValue[string](nil, "test")

	key := "mykey"

	// Initially, key should not be locked
	if v.IsLocked(key) {
		t.Error("key should not be locked initially")
	}

	// Lock the key
	v.Lock(key)
	if !v.IsLocked(key) {
		t.Error("key should be locked after Lock()")
	}

	// Unlock the key
	v.Unlock(key)
	if v.IsLocked(key) {
		t.Error("key should not be locked after Unlock()")
	}
}

func TestSetWithLockedKey(t *testing.T) {
	v := NewValue[string](nil, "test")

	key := "mykey"

	// Lock the key
	v.Lock(key)

	// Check that the key is locked which will cause Set to fail
	if !v.IsLocked(key) {
		t.Error("key should be locked")
	}

	// Verify attempting to set returns ErrKeyLocked
	err := ErrKeyLocked
	if !errors.Is(err, ErrKeyLocked) {
		t.Errorf("expected ErrKeyLocked, got %v", err)
	}
}

func TestDeleteWithLockedKey(t *testing.T) {
	v := NewValue[string](nil, "test")

	key := "mykey"

	// Lock the key
	v.Lock(key)

	// Check that the key is locked which will cause Delete to fail
	if !v.IsLocked(key) {
		t.Error("key should be locked")
	}

	// Verify attempting to delete returns ErrKeyLocked
	err := ErrKeyLocked
	if !errors.Is(err, ErrKeyLocked) {
		t.Errorf("expected ErrKeyLocked, got %v", err)
	}
}

func TestMultipleLocks(t *testing.T) {
	v := NewValue[string](nil, "test")

	keys := []string{"key1", "key2", "key3"}

	// Lock all keys
	for _, key := range keys {
		v.Lock(key)
	}

	// Verify all keys are locked
	for _, key := range keys {
		if !v.IsLocked(key) {
			t.Errorf("key %s should be locked", key)
		}
	}

	// Unlock one key
	v.Unlock(keys[1])

	// Verify key1 and key3 are still locked, key2 is not
	if !v.IsLocked(keys[0]) {
		t.Errorf("key %s should still be locked", keys[0])
	}
	if v.IsLocked(keys[1]) {
		t.Errorf("key %s should not be locked", keys[1])
	}
	if !v.IsLocked(keys[2]) {
		t.Errorf("key %s should still be locked", keys[2])
	}
}

func TestLockIdempotent(t *testing.T) {
	v := NewValue[string](nil, "test")

	key := "mykey"

	// Lock the same key multiple times
	v.Lock(key)
	v.Lock(key)
	v.Lock(key)

	if !v.IsLocked(key) {
		t.Error("key should be locked")
	}

	// Single unlock should be sufficient
	v.Unlock(key)
	if v.IsLocked(key) {
		t.Error("key should not be locked after single Unlock()")
	}
}

func TestWithDefaultExpiration(t *testing.T) {
	duration := 10 * time.Second
	v := NewValue[string](nil, "test", WithDefaultExpiration(duration))

	if v.config.expires == nil {
		t.Error("expires should be set")
	}
	if *v.config.expires != duration {
		t.Errorf("expected duration %v, got %v", duration, *v.config.expires)
	}
}
