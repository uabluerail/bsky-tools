package didset

import (
	"context"
)

type StringSet map[string]bool

type DIDSet interface {
	GetDIDs(ctx context.Context) (StringSet, error)
}

func (set StringSet) Clone() StringSet {
	r := StringSet{}
	for s := range set {
		r[s] = true
	}
	return r
}

type QueryableDIDSet interface {
	DIDSet
	Contains(ctx context.Context, did string) (bool, error)
}

type constSet struct {
	entries StringSet
}

func Const(dids ...string) QueryableDIDSet {
	r := &constSet{entries: StringSet{}}
	for _, did := range dids {
		r.entries[did] = true
	}
	return r
}

func (c *constSet) GetDIDs(ctx context.Context) (StringSet, error) {
	return c.entries.Clone(), nil
}

func (c *constSet) Contains(ctx context.Context, did string) (bool, error) {
	return c.entries[did], nil
}
