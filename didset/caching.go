package didset

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog"

	"github.com/bluesky-social/indigo/xrpc"
)

type caching struct {
	source DIDSet

	mu             sync.Mutex
	entries        StringSet
	waitUntilReady chan struct{}
	err            error
}

func (c *caching) run(ctx context.Context, refresh time.Duration) {
	log := zerolog.Ctx(ctx).With().Str("module", "didset").Logger()
	ctx = log.WithContext(ctx)

	waitChanClosed := false

	t := time.NewTicker(refresh)
	tr := make(chan time.Time, 1)
	tr <- time.Now()
	for {
		select {
		case <-tr:
			set, err := c.source.GetDIDs(ctx)
			if err != nil {
				log.Warn().Err(err).Msgf("Failed to refresh cached list")
				c.mu.Lock()
				c.err = err
				c.mu.Unlock()

				var xrpcErr *xrpc.Error
				if errors.As(err, &xrpcErr) {
					// If we're throttled - wake up and retry after ratelimit reset.
					if xrpcErr.IsThrottled() && xrpcErr.Ratelimit != nil {
						d := time.Until(xrpcErr.Ratelimit.Reset)
						go func() {
							time.Sleep(d)
							tr <- time.Now()
						}()
					}
				}
				break
			}
			c.mu.Lock()
			c.entries = set
			c.err = nil
			c.mu.Unlock()
			if !waitChanClosed {
				close(c.waitUntilReady)
				waitChanClosed = true
			}
		case v := <-t.C:
			tr <- v
		case <-ctx.Done():
			c.mu.Lock()
			defer c.mu.Unlock()
			c.err = fmt.Errorf("context of the background goroutine is done: %w", ctx.Err())
			return
		}
	}
}

func (c *caching) GetDIDs(ctx context.Context) (StringSet, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-c.waitUntilReady:
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	return c.entries.Clone(), c.err
}

func (c *caching) Contains(ctx context.Context, did string) (bool, error) {
	select {
	case <-ctx.Done():
		return false, ctx.Err()
	case <-c.waitUntilReady:
	}

	c.mu.Lock()
	if c.err != nil {
		defer c.mu.Unlock()
		return false, c.err
	}
	r := c.entries[did]
	c.mu.Unlock()

	return r, nil
}

func Cached(ctx context.Context, refresh time.Duration, source DIDSet) QueryableDIDSet {
	r := &caching{
		entries:        StringSet{},
		source:         source,
		waitUntilReady: make(chan struct{}),
	}
	go r.run(ctx, refresh)
	return r
}

func CachedIndividually(ctx context.Context, refresh time.Duration, sources ...DIDSet) []DIDSet {
	r := []DIDSet{}
	for _, source := range sources {
		r = append(r, Cached(ctx, refresh, source))
	}
	return r
}
