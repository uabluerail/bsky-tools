package aturl

import (
	"net/url"
	"strings"
)

func Parse(u string) (*url.URL, error) {
	if !strings.HasPrefix(u, "at://") {
		return url.Parse(u)
	}
	parts := strings.SplitN(strings.TrimPrefix(u, "at://"), "/", 2)
	r := &url.URL{
		Scheme: "at",
		Host:   parts[0],
	}

	if len(parts) > 1 {
		return r.Parse(parts[1])
	}

	return r, nil
}
