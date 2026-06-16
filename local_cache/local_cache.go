package localcache

import "sync"

type LocalCache[T comparable, V any] struct {
	data map[T]V
	mu   sync.RWMutex
}

// Create a new safe map
//
// Usage: m := safemap.New[keyType, valueType]()
func New[T comparable, V any]() *LocalCache[T, V] {
	return &LocalCache[T, V]{data: make(map[T]V)}
}

// Set a value in the map
//
// Usage: m.Set(key, value)
func (m *LocalCache[T, V]) Set(key T, value V) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.data == nil {
		m.data = make(map[T]V)
	}

	m.data[key] = value
}

// Get a value from the map by key
//
// Usage: val, ok := m.Get(key)
func (m *LocalCache[T, V]) Get(key T) (V, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	val, ok := m.data[key]

	return val, ok
}

// Delete a value from the map by key
//
// Usage: m.Delete(key)
func (m *LocalCache[T, V]) Delete(key T) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.data, key)
}

// Clear the map
//
// Usage: m.Clear()
func (m *LocalCache[T, V]) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.data = make(map[T]V)
}

