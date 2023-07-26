package didset

import "context"

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
