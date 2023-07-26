package didset

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	comatproto "github.com/bluesky-social/indigo/api/atproto"
	"github.com/bluesky-social/indigo/api/bsky"
	"github.com/bluesky-social/indigo/xrpc"
)

type blocked struct {
	client *xrpc.Client
}

func (b *blocked) GetDIDs(ctx context.Context) (StringSet, error) {
	log := zerolog.Ctx(ctx).With().
		Str("module", "didset").
		Str("didset", "blocked").
		Logger()
	ctx = log.WithContext(ctx)

	r := StringSet{}

	resp, err := comatproto.ServerGetSession(ctx, b.client)
	if err != nil {
		return nil, fmt.Errorf("ServerGetSession: %w", err)
	}
	self := resp.Did

	cursor := ""
	for {
		resp, err := comatproto.RepoListRecords(ctx, b.client, "app.bsky.graph.block", cursor, 100, self, false, "", "")
		if err != nil {
			return nil, fmt.Errorf("listing blocked users: %w", err)
		}
		if resp.Cursor == nil || *resp.Cursor == "" {
			break
		}
		cursor = *resp.Cursor
		for _, rec := range resp.Records {
			item, ok := rec.Value.Val.(*bsky.GraphBlock)
			if !ok {
				continue
			}

			r[item.Subject] = true
		}
	}

	log.Debug().Msgf("Got %d dids", len(r))
	return r, nil
}

func BlockedUsers(authclient *xrpc.Client) DIDSet {
	return &blocked{client: authclient}
}

type muteList struct {
	client *xrpc.Client
	url    string
}

func (l *muteList) GetDIDs(ctx context.Context) (StringSet, error) {
	log := zerolog.Ctx(ctx).With().
		Str("module", "didset").
		Str("didset", "mutelist").
		Str("list_url", l.url).
		Logger()
	ctx = log.WithContext(ctx)

	r := StringSet{}
	cursor := ""
	for {
		resp, err := bsky.GraphGetList(ctx, l.client, cursor, 100, l.url)
		if err != nil {
			return nil, fmt.Errorf("app.bsky.graph.getList: %w", err)
		}

		if len(resp.Items) == 0 {
			break
		}

		for _, item := range resp.Items {
			r[item.Subject.Did] = true
		}

		if resp.Cursor == nil {
			break
		}
		cursor = *resp.Cursor
	}
	log.Debug().Msgf("Got %d dids", len(r))

	return r, nil
}

func MuteList(authclient *xrpc.Client, url string) DIDSet {
	return &muteList{client: authclient, url: url}
}

type followers struct {
	client *xrpc.Client
	did    string
}

func (f *followers) GetDIDs(ctx context.Context) (StringSet, error) {
	log := zerolog.Ctx(ctx).With().
		Str("module", "didset").
		Str("didset", "followers").
		Str("followers_of", f.did).
		Logger()
	ctx = log.WithContext(ctx)

	r := StringSet{}
	cursor := ""
	for {
		resp, err := bsky.GraphGetFollowers(ctx, f.client, f.did, cursor, 100)
		if err != nil {
			return nil, fmt.Errorf("app.bsky.graph.getFollowers: %w", err)
		}

		if len(resp.Followers) == 0 {
			break
		}

		for _, item := range resp.Followers {
			r[item.Did] = true
		}

		if resp.Cursor == nil {
			break
		}
		cursor = *resp.Cursor
	}
	log.Debug().Msgf("Got %d dids", len(r))

	return r, nil
}

func FollowersOf(authclient *xrpc.Client, did string) DIDSet {
	return &followers{client: authclient, did: did}
}
