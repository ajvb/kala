package job

import "github.com/ajvb/kala/v2/clock"

// clockProvider is implemented by caches that carry a clock. A job takes the
// clock of the cache it is added to.
type clockProvider interface {
	Clock() clock.Clock
}

// clockOf returns the clock of the cache, or the real clock if the cache does
// not carry one.
func clockOf(cache JobCache) clock.Clock {
	if p, ok := cache.(clockProvider); ok {
		return p.Clock()
	}
	return clock.Real{}
}

// CacheOption configures a cache when it is created.
type CacheOption func(*cacheOptions)

type cacheOptions struct {
	clock     clock.Clock
	cmdRunner CommandRunner
}

// WithClock makes the cache, and every job added to it, use clk instead of the
// real clock. Tests pass a clock.Fake. A nil clk is ignored.
func WithClock(clk clock.Clock) CacheOption {
	return func(o *cacheOptions) {
		if clk != nil {
			o.clock = clk
		}
	}
}

func newCacheOptions(opts []CacheOption) cacheOptions {
	o := cacheOptions{clock: clock.Real{}, cmdRunner: ExecRunner{}}
	for _, opt := range opts {
		opt(&o)
	}
	return o
}
