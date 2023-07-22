package automute

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/rs/zerolog"

	comatproto "github.com/bluesky-social/indigo/api/atproto"
	"github.com/bluesky-social/indigo/api/bsky"
	lexutil "github.com/bluesky-social/indigo/lex/util"
	"github.com/bluesky-social/indigo/xrpc"
)

type List struct {
	url        url.URL
	authclient *http.Client

	CheckResultExpiration time.Duration
	ListRefreshInterval   time.Duration
	Callback              func(ctx context.Context, client *xrpc.Client, did string) (bool, error)

	mu                 sync.Mutex
	existingEntries    map[string]bool
	negativeCheckCache map[string]time.Time

	checkQueue chan string
}

func New(url *url.URL, authclient *http.Client) *List {
	return &List{
		url:                   *url,
		authclient:            authclient,
		existingEntries:       map[string]bool{},
		negativeCheckCache:    map[string]time.Time{},
		CheckResultExpiration: 24 * time.Hour,
		ListRefreshInterval:   30 * time.Minute,
		checkQueue:            make(chan string, 50),
	}
}

func (l *List) Run(ctx context.Context) error {
	log := zerolog.Ctx(ctx).With().
		Str("module", "automute").
		Str("list_did", l.url.String()).
		Logger()
	ctx = log.WithContext(ctx)
	client := getXrpcClient(l.authclient)

	for {
		err := l.refreshList(ctx, client)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to refresh the list %q", l.url.String())
			time.Sleep(5 * time.Second)
			continue
		}
		break
	}

	refresh := time.NewTicker(l.ListRefreshInterval)
	for {
		// TODO: prune negativeCheckCache
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-refresh.C:
			err := l.refreshList(ctx, client)
			if err != nil {
				log.Error().Err(err).Msgf("Failed to refresh the list %q", l.url.String())
			}
		case did := <-l.checkQueue:
			skip := func(did string) bool {
				l.mu.Lock()
				defer l.mu.Unlock()
				if l.existingEntries[did] {
					return true
				}
				if time.Since(l.negativeCheckCache[did]) < l.CheckResultExpiration {
					return true
				}
				return false
			}(did)

			if skip {
				break
			}

			func(did string) {
				if l.Callback == nil {
					return
				}
				add, err := l.Callback(ctx, client, did)
				if err != nil {
					log.Error().Err(err).Msgf("Failed to check if a user should be added to the list")
				}

				if add {
					resp, err := comatproto.RepoCreateRecord(ctx, client, &comatproto.RepoCreateRecord_Input{
						Collection: "app.bsky.graph.listitem",
						Repo:       l.url.Host,
						Record: &lexutil.LexiconTypeDecoder{Val: &bsky.GraphListitem{
							List:      l.url.String(),
							Subject:   did,
							CreatedAt: time.Now().UTC().Format(time.RFC3339),
						}},
					})
					if err != nil {
						log.Error().Err(err).Msgf("Failed to add %q to the list %s", did, l.url.String())
						return
					}
					log.Debug().Msgf("Added %q to the list %s, cid=%s", did, l.url.String(), resp.Cid)
				}

				l.mu.Lock()
				if add {
					l.existingEntries[did] = true
				} else {
					l.negativeCheckCache[did] = time.Now()
				}
				l.mu.Unlock()
			}(did)
		}
	}
	return ctx.Err()
}

func (l *List) refreshList(ctx context.Context, client *xrpc.Client) error {
	cursor := ""
	entries := map[string]bool{}
	for {
		resp, err := bsky.GraphGetList(ctx, client, cursor, 100, l.url.String())
		if err != nil {
			return fmt.Errorf("app.bsky.graph.getList: %w", err)
		}

		if len(resp.Items) == 0 {
			break
		}

		for _, item := range resp.Items {
			entries[item.Subject.Did] = true
		}

		if resp.Cursor == nil {
			break
		}
		cursor = *resp.Cursor
	}
	l.mu.Lock()
	l.existingEntries = entries
	l.mu.Unlock()
	return nil
}

func (l *List) Check(did string) {
	defer recover()
	l.checkQueue <- did
}

func getXrpcClient(c *http.Client) *xrpc.Client {
	return &xrpc.Client{
		Client: c,
		Host:   "https://bsky.social",
	}
}
