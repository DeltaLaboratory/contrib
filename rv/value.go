package rv

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"time"

	"github.com/fxamacker/cbor/v2"
	"github.com/redis/rueidis"
	"github.com/redis/rueidis/rueidislock"
)

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

func WithDefaultExpiration(duration time.Duration) Option {
	return func(r *valueConfig) {
		r.expires = &duration
	}
}

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

func (r *Value[T]) Get(ctx context.Context, key string) (*T, error) {
	var value T

	resp, err := r.client.Do(ctx, r.client.B().Get().Key(r.key+":"+key).Build()).AsBytes()
	if err != nil {
		if rueidis.IsRedisNil(err) {
			return nil, nil // Key does not exist
		}
		return nil, fmt.Errorf("failed to get value: %w", err)
	}

	if resp == nil {
		return nil, errors.New("value not found")
	}

	err = cbor.Unmarshal(resp, &value)
	if err != nil {
		return nil, fmt.Errorf("failed to decode value: %w", err)
	}

	return &value, nil
}

func (r *Value[T]) Delete(ctx context.Context, key string) error {
	err := r.client.Do(ctx, r.client.B().Del().Key(r.key+":"+key).Build()).Error()
	if err != nil {
		return fmt.Errorf("failed to delete value: %w", err)
	}

	return nil
}
