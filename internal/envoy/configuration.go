package envoy

import (
	"strconv"
	"sync/atomic"

	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
)

type GenericConfiguration struct {
	Version   int64
	Resources map[string][]types.Resource
}

func (g *GenericConfiguration) IncrementVersion() {
	atomic.AddInt64(&g.Version, 1)
}

func (g *GenericConfiguration) DoSnapshotCache() (*cache.Snapshot, error) {

	snapshot, err := cache.NewSnapshot(
		strconv.FormatInt(atomic.LoadInt64(&g.Version), 10),
		g.Resources,
	)

	if err != nil {
		return nil, err
	}

	if err := snapshot.Consistent(); err != nil {
		return nil, err
	}

	return snapshot, nil

}
