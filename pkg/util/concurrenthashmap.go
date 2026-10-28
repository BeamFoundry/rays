package util

import (
	"iter"
	"sync"

	. "github.com/BeamFoundry/rays/pkg/core"
	. "github.com/BeamFoundry/rays/pkg/lang"
)

var _ Map[string, any] = &ConcurrentHashMap[string, any]{}

// ConcurrentHashMap is a [Map] implementation that is thread-safe
//
// Internally it's just a [HasmMap] combined to a [sync.RWMutex]
//
// TODO: Review lock.
type ConcurrentHashMap[K comparable, V any] struct {
	m  HashMap[K, V]
	mu *sync.RWMutex
}

// New() creates a new ConcurrentHashMap
func (this ConcurrentHashMap[K, V]) New() *ConcurrentHashMap[K, V] {
	this.m = NewHashMap[K, V]()
	this.mu = &sync.RWMutex{}
	return &this
}

// Map() implements [Map.Map]
func (this *ConcurrentHashMap[K, V]) Map() map[K]V {
	this.mu.RLock()
	defer this.mu.RUnlock()
	return this.m.Map()
}

// Clear() implements [Map.Clear]
func (this *ConcurrentHashMap[K, V]) Clear() {
	this.mu.Lock()
	defer this.mu.Unlock()
	this.m.Clear()
}

// ContainsKey() implements [Map.ContainsKey]
func (this *ConcurrentHashMap[K, V]) ContainsKey(key K) bool {
	this.mu.RLock()
	defer this.mu.RUnlock()
	return this.m.ContainsKey(key)
}

// Merge() implements [Map.Merge].
func (this *ConcurrentHashMap[K, V]) Merge(m Map[K, V]) {
	this.mu.Lock()
	defer this.mu.Unlock()
	this.m.Merge(m)
}

// Iterator() implements [Map.Iterator]
func (this *ConcurrentHashMap[K, V]) Iterator() iter.Seq2[K, V] {
	this.mu.RLock()
	defer this.mu.RUnlock()
	return this.m.Iterator()
}

// Equals() implements [Map.Equals].
func (this *ConcurrentHashMap[K, V]) Equals(m Map[K, V]) bool {
	this.mu.RLock()
	defer this.mu.RUnlock()
	return this.m.Equals(m)
}

// Get() implements [Map.Get]
func (this *ConcurrentHashMap[K, V]) Get(key K) Optional[V] {
	this.mu.RLock()
	defer this.mu.RUnlock()
	return this.m.Get(key)
}

// Remove() implements [Map.Remove]
func (this *ConcurrentHashMap[K, V]) Remove(key K) Optional[V] {
	this.mu.Lock()
	defer this.mu.Unlock()
	return this.m.Remove(key)
}

// Delete() implements [Map.Delete]
func (this *ConcurrentHashMap[K, V]) Delete(f func(key K, val V) bool) {
	this.mu.Lock()
	defer this.mu.Unlock()
	this.m.Delete(f)
}

// IsEmpty() implements [Map.IsEmpty]
func (this *ConcurrentHashMap[K, V]) IsEmpty() bool {
	this.mu.RLock()
	defer this.mu.RUnlock()
	return this.m.IsEmpty()
}

// Put() implements [Map.Put]
func (this *ConcurrentHashMap[K, V]) Put(key K, val V) Map[K, V] {
	this.mu.Lock()
	defer this.mu.Unlock()
	return this.m.Put(key, val)
}

// PutAll() implements [Map.PutAll]
func (this *ConcurrentHashMap[K, V]) PutAll(m Map[K, V]) Map[K, V] {
	this.mu.Lock()
	defer this.mu.Unlock()
	return this.m.PutAll(m)
}

// Size() implements [Map.Size]
func (this *ConcurrentHashMap[K, V]) Size() int {
	this.mu.RLock()
	defer this.mu.RUnlock()
	return this.m.Size()
}

// MashalJSON() implements [json.Marshaler]
func (this *ConcurrentHashMap[K, V]) MarshalJSON() ([]byte, error) {
	this.mu.RLock()
	defer this.mu.RUnlock()
	return this.m.MarshalJSON()
}

// UnmarshalJSON() implements [json.Unmarshal]
func (this *ConcurrentHashMap[K, V]) UnmarshalJSON(data []byte) error {
	this.mu.Lock()
	defer this.mu.Unlock()
	return this.m.UnmarshalJSON(data)
}
