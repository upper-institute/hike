package envoy

import (
	"bytes"
	"crypto/sha256"
	"strconv"
	"sync/atomic"

	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"google.golang.org/protobuf/encoding/protojson"
)

type GenericConfiguration struct {
	Version       int64
	Resources     map[string][]types.Resource
	ResourcesHash []byte
	LastSnapshot  *cache.Snapshot
}

func (g *GenericConfiguration) Reset() {
	g.Resources = make(map[string][]types.Resource)
}

func (g *GenericConfiguration) IncrementVersion() {
	atomic.AddInt64(&g.Version, 1)
}

func (g *GenericConfiguration) HashResources() []byte {

	hash := sha256.New()

	for _, resources := range g.Resources {

		for _, resource := range resources {

			msg, err := protojson.Marshal(resource)
			if err != nil {
				panic(err)
			}

			hash.Write(msg)

		}

	}

	return hash.Sum(nil)

}

func (g *GenericConfiguration) DoSnapshotCache() (*cache.Snapshot, error) {

	newHash := g.HashResources()

	if g.ResourcesHash != nil && bytes.Equal(newHash, g.ResourcesHash) {
		return g.LastSnapshot, nil
	}

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

	g.LastSnapshot = snapshot
	g.ResourcesHash = newHash
	g.IncrementVersion()

	return snapshot, nil

}
