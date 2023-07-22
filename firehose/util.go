package firehose

import (
	"fmt"
	"strings"

	comatproto "github.com/bluesky-social/indigo/api/atproto"
)

func FeedPostURL(commit *comatproto.SyncSubscribeRepos_Commit, op *comatproto.SyncSubscribeRepos_RepoOp) string {
	repo := commit.Repo
	id := strings.TrimPrefix(op.Path, strings.Split(op.Path, "/")[0]+"/")
	return fmt.Sprintf("https://bsky.app/profile/%s/post/%s", repo, id)
}
