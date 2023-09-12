package xrpcauth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"golang.org/x/oauth2"

	comatproto "github.com/bluesky-social/indigo/api/atproto"
	"github.com/bluesky-social/indigo/util"
	"github.com/bluesky-social/indigo/xrpc"
	"github.com/golang-jwt/jwt/v5"
)

type FileBackedTokenSource struct {
	filename string
}

func (s *FileBackedTokenSource) Token() (*oauth2.Token, error) {
	b, err := os.ReadFile(s.filename)
	if err != nil {
		return nil, fmt.Errorf("reading token file %q: %w", s.filename, err)
	}
	tok := &comatproto.ServerRefreshSession_Output{}
	if err := json.Unmarshal(b, tok); err != nil {
		return nil, fmt.Errorf("unmarshaling stored token: %w", err)
	}

	client := &xrpc.Client{
		Client: util.RobustHTTPClient(),
		Host:   "https://bsky.social",
		Auth: &xrpc.AuthInfo{
			AccessJwt: tok.RefreshJwt,
			Handle:    tok.Handle,
			Did:       tok.Did,
		},
	}

	refresh, err := comatproto.ServerRefreshSession(context.Background(), client)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh the token: %w", err)
	}

	b, err = json.Marshal(refresh)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal the refreshed token: %w", err)
	}
	if err := os.WriteFile(s.filename+".tmp", b, 0600); err != nil {
		return nil, fmt.Errorf("failed to write the token into a temp file: %w", err)
	}
	if err := os.Rename(s.filename+".tmp", s.filename); err != nil {
		return nil, fmt.Errorf("failed to replace the token file: %w", err)
	}

	t, _, err := jwt.NewParser().ParseUnverified(refresh.AccessJwt, make(jwt.MapClaims))
	if err != nil {
		return nil, fmt.Errorf("failed to parse access token: %w", err)
	}
	expiry, err := t.Claims.GetExpirationTime()
	if err != nil {
		return nil, fmt.Errorf("failed to get expiration time from the access token: %w", err)
	}

	r := &oauth2.Token{
		TokenType:    "bearer",
		AccessToken:  refresh.AccessJwt,
		RefreshToken: refresh.RefreshJwt,
		Expiry:       expiry.Time,
	}

	return r, nil
}

func NewHttpClient(ctx context.Context, authfile string) *http.Client {
	ctx = context.WithValue(ctx, oauth2.HTTPClient, util.RobustHTTPClient())
	return oauth2.NewClient(ctx, &FileBackedTokenSource{filename: authfile})
}

func NewClient(ctx context.Context, authfile string) *xrpc.Client {
	return &xrpc.Client{
		Client: NewHttpClient(ctx, authfile),
		Host:   "https://bsky.social",
	}
}
