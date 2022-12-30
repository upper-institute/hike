package envoy

import (
	"bytes"
	"crypto/sha256"
	"strconv"
	"sync/atomic"

	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"
)

type Configuration struct {
	version      int64
	stateHash    []byte
	lastSnapshot *cache.Snapshot

	logger *zap.SugaredLogger

	Resources map[string][]types.Resource
}

func NewConfiguration(logger *zap.SugaredLogger) *Configuration {
	return &Configuration{
		logger: logger,
	}
}

func (c *Configuration) hashResources() []byte {

	hash := sha256.New()

	for _, resources := range c.Resources {

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

func (c *Configuration) DoSnapshot() (*cache.Snapshot, error) {

	newHash := c.hashResources()

	if c.stateHash != nil && bytes.Equal(newHash, c.stateHash) {
		return c.lastSnapshot, nil
	}

	snapshot, err := cache.NewSnapshot(
		strconv.FormatInt(atomic.LoadInt64(&c.version), 10),
		c.Resources,
	)
	if err != nil {
		return nil, err
	}

	if err := snapshot.Consistent(); err != nil {
		return nil, err
	}

	c.lastSnapshot = snapshot
	c.stateHash = newHash
	atomic.AddInt64(&c.version, 1)

	return snapshot, nil

}
