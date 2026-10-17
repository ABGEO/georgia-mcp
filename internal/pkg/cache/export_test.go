package cache

// EntryCount returns the count of entries in the store, after the store removes
// the entries past its size limit.
func EntryCount[V any](c *Cache[V]) int {
	c.store.CleanUp()

	return c.store.EstimatedSize()
}
