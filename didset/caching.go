package didset

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

type caching struct {
	source DIDSet

	mu       sync.Mutex
	entries  StringSet
	haveData bool
}

func (c *caching) run(ctx context.Context, refresh time.Duration) {
	log := zerolog.Ctx(ctx).With().Str("module", "didset").Logger()
	ctx = log.WithContext(ctx)

	t := time.NewTicker(refresh)
	tr := make(chan time.Time, 1)
	tr <- time.Now()
	for {
		select {
		case <-tr:
			set, err := c.source.GetDIDs(ctx)
			if err != nil {
				log.Warn().Err(err).Msgf("Failed to refresh cached list")
				break
			}
			c.mu.Lock()
			c.entries = set
			c.haveData = true
			c.mu.Unlock()
		case v := <-t.C:
			tr <- v
		case <-ctx.Done():
			c.mu.Lock()
			defer c.mu.Unlock()
			c.haveData = false
			return
		}
	}
}

func (c *caching) GetDIDs(ctx context.Context) (StringSet, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.haveData {
		return nil, fmt.Errorf("cache not populated yet")
	}

	return c.entries.Clone(), nil
}

func Cached(ctx context.Context, refresh time.Duration, source DIDSet) DIDSet {
	r := &caching{entries: StringSet{}, source: source}
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
