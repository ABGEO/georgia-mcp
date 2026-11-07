package cache

import "context"

// EntryCount returns the count of entries in the store, after the store removes
// the entries past its size limit.
func EntryCount[V any](c *Cache[V]) int {
	c.store.CleanUp()

	return c.store.EstimatedSize()
}

// SharedLoad runs the shared load of the key, as a caller that missed the store does.
func SharedLoad[V any](
	ctx context.Context,
	c *Cache[V],
	key string,
	load func(context.Context) (V, error),
) (V, error) {
	loaded, err := c.load(ctx, key, load)

	return loaded.value, err
}

// SharedLoadMany runs the shared load of the keys, as a caller that missed the store does.
func SharedLoadMany[V any](
	ctx context.Context,
	c *Cache[V],
	keys []string,
	load func(context.Context, []string) (map[string]V, error),
) (map[string]V, error) {
	loaded, err := c.loadMany(ctx, keys, load)
	if err != nil {
		return nil, err
	}

	values := make(map[string]V, len(loaded))
	for key, value := range loaded {
		values[key] = value.value
	}

	return values, nil
}
