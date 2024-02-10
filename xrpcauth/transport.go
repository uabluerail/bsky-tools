package xrpcauth

import (
	"net/http"
	"net/url"

	"github.com/bluesky-social/indigo/xrpc"
)

func stripAuthHeaderOnRedirect(client *xrpc.Client) http.RoundTripper {
	return &stripAuthTransport{
		client: client,
		base:   http.DefaultTransport.(*http.Transport).Clone(),
	}
}

type stripAuthTransport struct {
	client *xrpc.Client
	base   http.RoundTripper
}

// RoundTrip authorizes and authenticates the request with an
// access token from Transport's Source.
func (t *stripAuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	reqBodyClosed := false
	if req.Body != nil {
		defer func() {
			if !reqBodyClosed {
				req.Body.Close()
			}
		}()
	}

	req2 := req.Clone(req.Context())
	host := "bsky.social"
	if u, err := url.Parse(t.client.Host); err == nil {
		host = u.Host
	}
	// Hack for supporting redirects during a transition to split-PDS deployment
	// (https://github.com/bluesky-social/atproto/discussions/1832)
	// golang.org/x/oauth2 lib blindly adds `Authorization` header on redirects,
	// which is inconsistent with http.Client behaviour and can potentially leak the token.
	// But more immediate issue is that it can make some requests fail due to the redirected
	// endpoint not being able to validate the token, even if it normally doesn't
	// require any auth.
	reqHost := req2.Host
	if reqHost == "" {
		reqHost = req2.URL.Host
	}
	if reqHost != host {
		req2.Header.Del("Authorization")
	}

	// req.Body is assumed to be closed by the base RoundTripper.
	reqBodyClosed = true
	return t.base.RoundTrip(req2)
}
