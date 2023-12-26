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

func (set StringSet) GetDIDs(ctx context.Context) (StringSet, error) {
	return set.Clone(), nil
}

func (set StringSet) Contains(ctx context.Context, did string) (bool, error) {
	return set[did], nil
}

type QueryableDIDSet interface {
	DIDSet
	Contains(ctx context.Context, did string) (bool, error)
}

func Const(dids ...string) QueryableDIDSet {
	r := StringSet{}
	for _, did := range dids {
		r[did] = true
	}
	return r
}
