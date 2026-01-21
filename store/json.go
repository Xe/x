package store

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

func z[T any]() T { return *new(T) }

type JSON[T any] struct {
	Underlying Interface
	Prefix     string
}

func (j *JSON[T]) Delete(ctx context.Context, key string) error {
	if j.Prefix != "" {
		key = j.Prefix + "/" + key
	}

	return j.Underlying.Delete(ctx, key)
}

func (j *JSON[T]) Exists(ctx context.Context, key string) error {
	if j.Prefix != "" {
		key = j.Prefix + "/" + key
	}

	return j.Underlying.Exists(ctx, key)
}

func (j *JSON[T]) Get(ctx context.Context, key string) (T, error) {
	if j.Prefix != "" {
		key = j.Prefix + "/" + key
	}

	data, err := j.Underlying.Get(ctx, key)
	if err != nil {
		return z[T](), err
	}

	var result T
	if err := json.Unmarshal(data, &result); err != nil {
		return z[T](), fmt.Errorf("%w: %w", ErrCantDecode, err)
	}

	return result, nil
}

func (j *JSON[T]) Set(ctx context.Context, key string, value T) error {
	if j.Prefix != "" {
		key = j.Prefix + "/" + key
	}

	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrCantEncode, err)
	}

	if err := j.Underlying.Set(ctx, key, data); err != nil {
		return err
	}

	return nil
}

func (j *JSON[T]) List(ctx context.Context, prefix string) ([]string, error) {
	fullPrefix := j.Prefix + "/" + prefix
	keys, err := j.Underlying.List(ctx, fullPrefix)
	if err != nil {
		return nil, err
	}

	// Strip the full prefix from each key.
	result := make([]string, 0, len(keys))
	for _, k := range keys {
		result = append(result, strings.TrimPrefix(k, fullPrefix))
	}

	return result, nil
}
