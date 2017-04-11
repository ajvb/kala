package hashmap

import (
	"bytes"
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
	"unsafe"
)

// MaxFillRate is the maximum fill rate for the slice before a resize  will happen.
const MaxFillRate = 50

type (
	hashMapData struct {
		keyRightShifts uint64         // 64 - log2 of array size, to be used as index in the data array
		data           unsafe.Pointer // pointer to slice data array
		slice          []*ListElement // storage for the slice for the garbage collector to not clean it up
		count          uint64         // count of filled elements in the slice
	}

	// HashMap implements a read optimized hash map.
	HashMap struct {
		mapDataPtr unsafe.Pointer // pointer to a map instance that gets replaced if the map resizes
		linkedList *List          // key sorted linked list of elements
		sync.Mutex                // mutex that is only used for resize operations
	}

	// KeyValue represents a key/value that is returned by the iterator.
	KeyValue struct {
		Key   interface{}
		Value unsafe.Pointer
	}
)

// New returns a new HashMap.
func New() *HashMap {
	return NewSize(8)
}

// NewSize returns a new HashMap instance with a specific initialization size.
func NewSize(size uint64) *HashMap {
	hashmap := &HashMap{
		linkedList: NewList(),
	}
	hashmap.Grow(size)
	return hashmap
}

// Len returns the number of elements within the map.
func (m *HashMap) Len() uint64 {
	return m.linkedList.Len()
}

func (m *HashMap) mapData() *hashMapData {
	return (*hashMapData)(atomic.LoadPointer(&m.mapDataPtr))
}

// Fillrate returns the fill rate of the map as an percentage integer.
func (m *HashMap) Fillrate() uint64 {
	mapData := m.mapData()
	count := atomic.LoadUint64(&mapData.count)
	sliceLen := uint64(len(mapData.slice))
	return (count * 100) / sliceLen
}

func (m *HashMap) getSliceItemForKey(hashedKey uint64) (mapData *hashMapData, item *ListElement) {
	mapData = m.mapData()
	index := hashedKey >> mapData.keyRightShifts
	sliceDataIndexPointer := (*unsafe.Pointer)(unsafe.Pointer(uintptr(mapData.data) + uintptr(index*intSizeBytes)))
	item = (*ListElement)(atomic.LoadPointer(sliceDataIndexPointer))
	return
}

// Del deletes the hashed key from the map.
func (m *HashMap) Del(key interface{}) {
	hashedKey := getKeyHash(key)
	for _, entry := m.getSliceItemForKey(hashedKey); entry != nil; entry = entry.Next() {
		if entry.keyHash == hashedKey && entry.key == key {
			m.linkedList.Delete(entry)
			return
		}

		if entry.keyHash > hashedKey {
			return
		}
	}
}

// DelHashedKey deletes the hashed key from the map.
func (m *HashMap) DelHashedKey(hashedKey uint64) {
	for _, entry := m.getSliceItemForKey(hashedKey); entry != nil; entry = entry.Next() {
		if entry.keyHash == hashedKey {
			m.linkedList.Delete(entry)
			return
		}

		if entry.keyHash > hashedKey {
			return
		}
	}
}

// Insert sets the value under the specified key to the map if it does not exist yet.
// If a resizing operation is happening concurrently while calling Set, the item might show up in the map only after the resize operation is finished.
// Returns true if the item was inserted or false if it existed.
func (m *HashMap) Insert(key interface{}, value unsafe.Pointer) bool {
	hashedKey := getKeyHash(key)

	newEntry := &ListElement{
		key:     key,
		keyHash: hashedKey,
		value:   value,
	}

	return m.insertListElement(newEntry, false)
}

// Set sets the value under the specified key to the map. An existing item for this key will be overwritten.
// If a resizing operation is happening concurrently while calling Set, the item might show up in the map only after the resize operation is finished.
func (m *HashMap) Set(key interface{}, value unsafe.Pointer) {
	hashedKey := getKeyHash(key)

	newEntry := &ListElement{
		key:     key,
		keyHash: hashedKey,
		value:   value,
	}

	m.insertListElement(newEntry, true)
}

// SetHashedKey sets the value under the specified hash key to the map. An existing item for this key will be overwritten.
// You can use this function if your keys are already hashes and you want to avoid another hashing of the key.
// Do not use non hashes as keys for this function, the performance would decrease!
// If a resizing operation is happening concurrently while calling Set, the item might show up in the map only after the resize operation is finished.
func (m *HashMap) SetHashedKey(hashedKey uint64, value unsafe.Pointer) {
	newEntry := &ListElement{
		key:     hashedKey,
		keyHash: hashedKey,
		value:   value,
	}

	m.insertListElement(newEntry, true)
}

func (m *HashMap) insertListElement(newEntry *ListElement, update bool) bool {
	for {
		mapData, sliceItem := m.getSliceItemForKey(newEntry.keyHash)
		if update {
			if !m.linkedList.AddOrUpdate(newEntry, sliceItem) {
				continue // a concurrent add did interfere, try again
			}
		} else {
			existed, inserted := m.linkedList.Add(newEntry, sliceItem)
			if existed {
				return false
			}
			if !inserted {
				continue
			}
		}

		newSliceCount := mapData.addItemToIndex(newEntry)
		if newSliceCount != 0 {
			sliceLen := uint64(len(mapData.slice))
			fillRate := (newSliceCount * 100) / sliceLen

			if fillRate > MaxFillRate { // check if the slice needs to be resized
				m.Lock()
				currentMapData := m.mapData()
				if mapData == currentMapData { // double check that no other resize happened
					m.grow(0)
				}
				m.Unlock()
			}
		}
		return true
	}
}

// CasHashedKey performs a compare and swap operation sets the value under the specified hash key to the map. An existing item for this key will be overwritten.
func (m *HashMap) CasHashedKey(hashedKey uint64, from, to unsafe.Pointer) bool {
	newEntry := &ListElement{
		key:     hashedKey,
		keyHash: hashedKey,
		value:   to,
	}

	for {
		mapData, sliceItem := m.getSliceItemForKey(hashedKey)
		if !m.linkedList.Cas(newEntry, from, sliceItem) {
			return false
		}

		newSliceCount := mapData.addItemToIndex(newEntry)
		if newSliceCount != 0 {
			sliceLen := uint64(len(mapData.slice))
			fillRate := (newSliceCount * 100) / sliceLen

			if fillRate > MaxFillRate { // check if the slice needs to be resized
				m.Lock()
				currentMapData := m.mapData()
				if mapData == currentMapData { // double check that no other resize happened
					m.grow(0)
				}
				m.Unlock()
			}
		}
		return true
	}
}

// adds an item to the index if needed and returns the new item counter if it changed, otherwise 0
func (mapData *hashMapData) addItemToIndex(item *ListElement) uint64 {
	index := item.keyHash >> mapData.keyRightShifts
	sliceDataIndexPointer := (*unsafe.Pointer)(unsafe.Pointer(uintptr(mapData.data) + uintptr(index*intSizeBytes)))

	for { // loop until the smallest key hash is in the index
		sliceItem := (*ListElement)(atomic.LoadPointer(sliceDataIndexPointer)) // get the current item in the index
		if sliceItem == nil {                                                  // no item yet at this index
			if atomic.CompareAndSwapPointer(sliceDataIndexPointer, nil, unsafe.Pointer(item)) {
				return atomic.AddUint64(&mapData.count, 1)
			}
			continue // a new item was inserted concurrently, retry
		}

		if item.keyHash < sliceItem.keyHash {
			// the new item is the smallest for this index?
			if !atomic.CompareAndSwapPointer(sliceDataIndexPointer, unsafe.Pointer(sliceItem), unsafe.Pointer(item)) {
				continue // a new item was inserted concurrently, retry
			}
		}
		return 0
	}
}

// Grow resizes the hashmap to a new size, gets rounded up to next power of 2.
// To double the size of the hashmap use newSize 0.
func (m *HashMap) Grow(newSize uint64) {
	m.Lock()
	m.grow(newSize)
	m.Unlock()
}

func (m *HashMap) grow(newSize uint64) {
	mapData := m.mapData()
	if newSize == 0 {
		newSize = uint64(len(mapData.slice)) << 1
	} else {
		newSize = roundUpPower2(newSize)
	}

	newSlice := make([]*ListElement, newSize)
	header := (*reflect.SliceHeader)(unsafe.Pointer(&newSlice))

	newMapData := &hashMapData{
		keyRightShifts: 64 - log2(newSize),
		data:           unsafe.Pointer(header.Data), // use address of slice data storage
		slice:          newSlice,
	}

	m.fillIndexItems(newMapData) // initialize new index slice with longer keys

	atomic.StorePointer(&m.mapDataPtr, unsafe.Pointer(newMapData))

	m.fillIndexItems(newMapData) // make sure that the new index is up to date with the current state of the linked list
}

func (m *HashMap) fillIndexItems(mapData *hashMapData) {
	first := m.linkedList.First()
	item := first
	lastIndex := uint64(0)

	for item != nil {
		index := item.keyHash >> mapData.keyRightShifts
		if item == first || index != lastIndex { // store item with smallest hash key for every index
			if !item.Deleted() {
				mapData.addItemToIndex(item)
				lastIndex = index
			}
		}
		item = item.Next()
	}
}

// String returns the map as a string, only hashed keys are printed.
func (m *HashMap) String() string {
	buffer := bytes.NewBufferString("")
	buffer.WriteRune('[')

	first := m.linkedList.First()
	item := first

	for item != nil {
		if !item.Deleted() {
			if item != first {
				buffer.WriteRune(',')
			}
			fmt.Fprint(buffer, item.keyHash)
		}

		item = item.Next()
	}
	buffer.WriteRune(']')
	return buffer.String()
}

// Iter returns an iterator which could be used in a for range loop.
// The order of the items is sorted by hash keys.
func (m *HashMap) Iter() <-chan KeyValue {
	ch := make(chan KeyValue) // do not use a size here since items can get added during iteration

	go func() {
		item := m.linkedList.First()
		for item != nil {
			if !item.Deleted() {
				ch <- KeyValue{item.key, item.Value()}
			}
			item = item.Next()
		}
		close(ch)
	}()

	return ch
}
