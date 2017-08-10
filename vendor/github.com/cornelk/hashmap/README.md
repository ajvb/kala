# hashmap [![Build Status](https://travis-ci.org/cornelk/hashmap.svg?branch=master)](https://travis-ci.org/cornelk/hashmap) [![GoDoc](https://godoc.org/github.com/cornelk/hashmap?status.svg)](https://godoc.org/github.com/cornelk/hashmap) [![Go Report Card](https://goreportcard.com/badge/cornelk/hashmap)](https://goreportcard.com/report/github.com/cornelk/hashmap) [![codecov](https://codecov.io/gh/cornelk/hashmap/branch/master/graph/badge.svg)](https://codecov.io/gh/cornelk/hashmap)

## Overview

A Golang thread-safe HashMap optimized for fastest lock-free read access on 64 bit systems.

## Benchmarks

The benchmarks were run with Golang 1.8 on MacOS.

Reading from the hash map in a thread-safe way is as fast as reading from a standard Golang map in an unsafe way and 3 times faster than reading from the [Golang syncmap](https://github.com/golang/sync/tree/master/syncmap):

```
BenchmarkReadHashMapUint-8                	 2000000	      6614 ns/op
BenchmarkReadGoMapUintUnsafe-8            	 2000000	      6284 ns/op
BenchmarkReadGoMapUintMutex-8             	  300000	     48435 ns/op
BenchmarkReadGoSyncMapUint-8              	 1000000	     15541 ns/op
```

If the keys of the map are already hashes, no extra hashing needs to be done by the map:

```
BenchmarkReadHashMapHashedKey-8           	10000000	      1564 ns/op
```

Reading from the map while writes are happening:
```
BenchmarkReadHashMapWithWritesUint-8      	 2000000	      8714 ns/op
BenchmarkReadGoMapWithWritesUintMutex-8   	  100000	    172649 ns/op
BenchmarkReadGoSyncMapWithWritesUint-8    	  300000	     37157 ns/op
```

Pure Write performance without any reads:

```
BenchmarkWriteHashMapUint-8               	  100000	    225702 ns/op
BenchmarkWriteGoMapMutexUint-8            	  300000	     59474 ns/op
BenchmarkWriteGoSyncMapUint-8             	  100000	    143835 ns/op
```

## Technical details

* Technical design decisions have been made based on benchmarks that are stored in an external repository: [go-benchmark](https://github.com/cornelk/go-benchmark)

* The API uses [unsafe.Pointer](https://golang.org/pkg/unsafe/#Pointer) instead of the common interface{} for the values for faster speed when reading values.

* The library uses a sorted linked list and a slice as an index into that list.

* The Get() function contains helper functions that have been inlined manually until the Golang compiler will inline them automatically. Golang 1.9 will bring inlining optimizations.

* It optimizes the slice access by circumventing the Golang size check when reading from the slice. Once a slice is allocated, the size of it does not change.
  The library limits the index into the slice, therefor the Golang size check is obsolete. When the slice reaches a defined fill rate, a bigger slice is allocated
and all keys are recalculated and transferred into the new slice.

* The resize operation uses a lock to ensure that only one resize operation is happening. This way, no CPU and memory resources are wasted by multiple goroutines working on the resize.
