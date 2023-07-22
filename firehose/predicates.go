package firehose

import (
	"context"

	comatproto "github.com/bluesky-social/indigo/api/atproto"
	"github.com/bluesky-social/indigo/api/bsky"
	cbg "github.com/whyrusleeping/cbor-gen"
)

func AllOf(predicates ...Predicate) Predicate {
	return func(ctx context.Context, commit *comatproto.SyncSubscribeRepos_Commit, op *comatproto.SyncSubscribeRepos_RepoOp, record cbg.CBORMarshaler) bool {
		for _, p := range predicates {
			if !p(ctx, commit, op, record) {
				return false
			}
		}
		return true
	}
}

func AnyOf(predicates ...Predicate) Predicate {
	return func(ctx context.Context, commit *comatproto.SyncSubscribeRepos_Commit, op *comatproto.SyncSubscribeRepos_RepoOp, record cbg.CBORMarshaler) bool {
		for _, p := range predicates {
			if p(ctx, commit, op, record) {
				return true
			}
		}
		return false
	}
}

func MentionsDID(did string) Predicate {
	return func(ctx context.Context, commit *comatproto.SyncSubscribeRepos_Commit, op *comatproto.SyncSubscribeRepos_RepoOp, record cbg.CBORMarshaler) bool {
		rec, ok := record.(*bsky.FeedPost)
		if !ok {
			return false
		}
		for _, facet := range rec.Facets {
			for _, feature := range facet.Features {
				if mention := feature.RichtextFacet_Mention; mention != nil {
					if mention.Did == did {
						return true
					}
				}
			}
		}
		return false
	}
}
