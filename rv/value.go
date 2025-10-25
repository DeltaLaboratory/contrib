package rv

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/fxamacker/cbor/v2"
	"github.com/redis/rueidis"
	"github.com/redis/rueidis/rueidislock"
)

// Value is a typed wrapper around a namespaced Redis keyspace backed by rueidis.
type Value[T any] struct {
	client rueidis.Client
	locker rueidislock.Locker
	key    string

	config valueConfig
}

type valueConfig struct {
	expires         *time.Duration
	lockKeyMajority int32
}

type Option func(*valueConfig)

// NewValue instantiates a Value helper for the provided key prefix and client options.
func NewValue[T any](clientOption rueidis.ClientOption, key string, options ...Option) *Value[T] {
	client, err := rueidis.NewClient(clientOption)
	if err != nil {
		panic(fmt.Errorf("failed to create redis client: %w", err))
	}

	r := &Value[T]{key: key, client: client}

	if len(options) > 0 {
		for _, opt := range options {
			opt(&r.config)
		}
	}

	locker, err := rueidislock.NewLocker(rueidislock.LockerOption{
		ClientOption:   clientOption,
		NoLoopTracking: true,

		KeyMajority: r.config.lockKeyMajority,
	})
	if err != nil {
		panic(fmt.Errorf("failed to create redis locker: %w", err))
	}

	r.locker = locker

	runtime.AddCleanup(r, func(locker rueidislock.Locker) { locker.Close() }, r.locker)
	runtime.AddCleanup(r, func(client rueidis.Client) { client.Close() }, r.client)

	return r
}

// WithDefaultExpiration configures a default TTL applied to every Set call.
func WithDefaultExpiration(duration time.Duration) Option {
	return func(r *valueConfig) {
		r.expires = &duration
	}
}

// WithKeyMajority customizes the distributed lock key-majority quorum.
func WithKeyMajority(n int32) Option {
	return func(r *valueConfig) {
		r.lockKeyMajority = n
	}
}

// WithLock acquires a distributed lock for the given key and executes the provided function within the lock's context.
// It does not mean that the key itself is locked, but rather a namespaced lock based on the provided key.
func (r *Value[T]) WithLock(ctx context.Context, key string, fn func(ctx context.Context) error) error {
	ctx, release, err := r.locker.WithContext(ctx, r.key+":"+key)
	if err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	defer release()

	return fn(ctx)
}

// Set encodes and stores the provided value under the namespaced key.
func (r *Value[T]) Set(ctx context.Context, key string, value *T) error {
	encoded, err := cbor.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to encode value: %w", err)
	}

	builder := r.client.B().Set().Key(r.key + ":" + key).Value(rueidis.BinaryString(encoded))

	if r.config.expires != nil {
		builder.Ex(*r.config.expires)
	}

	err = r.client.Do(ctx, builder.Build()).Error()
	if err != nil {
		return fmt.Errorf("failed to set value: %w", err)
	}

	return nil
}

// Get loads a value by key; it returns nil when the key does not exist.
func (r *Value[T]) Get(ctx context.Context, key string) (*T, error) {
	resp, err := r.client.Do(ctx, r.client.B().Get().Key(r.key+":"+key).Build()).AsBytes()
	if err != nil {
		if errors.Is(err, rueidis.Nil) {
			return nil, nil // Key does not exist
		}
		return nil, fmt.Errorf("failed to get value: %w", err)
	}

	if resp == nil {
		return nil, errors.New("value not found")
	}

	return r.decodeValue(resp)
}

// Delete removes the namespaced key from Redis.
func (r *Value[T]) Delete(ctx context.Context, key string) error {
	err := r.client.Do(ctx, r.client.B().Del().Key(r.key+":"+key).Build()).Error()
	if err != nil {
		return fmt.Errorf("failed to delete value: %w", err)
	}

	return nil
}

// Scan iterates through the namespaced keys that match the provided pattern (without the namespace prefix)
// and returns the decoded values. Passing an empty pattern matches all keys in the namespace.
func (r *Value[T]) Scan(ctx context.Context, pattern string) ([]*T, error) {
	match := r.key + ":"
	if pattern == "" {
		match += "*"
	} else {
		match += pattern
	}

	var (
		cursor uint64
		values []*T
	)

	for {
		entry, err := r.client.Do(ctx, r.client.B().Scan().Cursor(cursor).Match(match).Build()).AsScanEntry()
		if err != nil {
			return nil, fmt.Errorf("failed to scan values matching %q: %w", pattern, err)
		}

		if len(entry.Elements) > 0 {
			batch, err := rueidis.MGet(r.client, ctx, entry.Elements)
			if err != nil {
				return nil, fmt.Errorf("failed to fetch scan batch for %q: %w", pattern, err)
			}

			for _, rawKey := range entry.Elements {
				relativeKey := strings.TrimPrefix(rawKey, r.key+":")

				msg, ok := batch[rawKey]
				if !ok {
					continue
				}

				data, err := msg.AsBytes()
				if err != nil {
					if errors.Is(err, rueidis.Nil) {
						continue // key disappeared between SCAN and MGET
					}
					return nil, fmt.Errorf("failed to load key %q: %w", relativeKey, err)
				}

				value, err := r.decodeValue(data)
				if err != nil {
					return nil, fmt.Errorf("failed to decode key %q: %w", relativeKey, err)
				}

				values = append(values, value)
			}
		}

		if entry.Cursor == 0 {
			break
		}

		cursor = entry.Cursor
	}

	return values, nil
}

// decodeValue transforms the CBOR payload into the generic type.
func (r *Value[T]) decodeValue(data []byte) (*T, error) {
	var value T
	if err := cbor.Unmarshal(data, &value); err != nil {
		return nil, fmt.Errorf("failed to decode value: %w", err)
	}
	return &value, nil
}
