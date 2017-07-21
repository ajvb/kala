package hashmap

import (
	"sync/atomic"
	"unsafe"
)

// ListElement is an element of the list.
type ListElement struct {
	nextElement unsafe.Pointer
	key         interface{}
	keyHash     uint64
	value       unsafe.Pointer
	deleted     uint64 // for the root and all deleted items this is set to 1
}

// Value returns the value of the list item.
func (e *ListElement) Value() unsafe.Pointer {
	return atomic.LoadPointer(&e.value)
}

// Deleted returns whether the item was deleted.
func (e *ListElement) Deleted() bool {
	return atomic.LoadUint64(&e.deleted) == 1
}

// Next returns the item on the right.
func (e *ListElement) Next() *ListElement {
	return (*ListElement)(atomic.LoadPointer(&e.nextElement))
}

// SetDeleted sets the deleted flag of the item.
func (e *ListElement) SetDeleted(deleted bool) bool {
	if deleted {
		return atomic.CompareAndSwapUint64(&e.deleted, 0, 1)
	}
	return atomic.CompareAndSwapUint64(&e.deleted, 1, 0)
}

// SetValue sets the value of the item.
func (e *ListElement) SetValue(value unsafe.Pointer) {
	atomic.StorePointer(&e.value, value)
}

// CasValue compares and swaps the values of the item.
func (e *ListElement) CasValue(from, to unsafe.Pointer) bool {
	return atomic.CompareAndSwapPointer(&e.value, from, to)
}
