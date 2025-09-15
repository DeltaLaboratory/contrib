package rv

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/fxamacker/cbor/v2"
	"github.com/redis/rueidis"
)

type Value[T any] struct {
	client rueidis.Client
	key    string

	config valueConfig
}

type valueConfig struct {
	expires *time.Duration
}

type Option func(*valueConfig)

func NewValue[T any](client rueidis.Client, key string, options ...Option) *Value[T] {
	r := &Value[T]{client: client, key: key}

	if len(options) > 0 {
		for _, opt := range options {
			opt(&r.config)
		}
	}

	return r
}

func WithDefaultExpiration(duration time.Duration) Option {
	return func(r *valueConfig) {
		r.expires = &duration
	}
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
