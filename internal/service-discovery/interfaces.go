package servicediscovery

import (
	"context"

	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
)

type ServiceDiscoveryService interface {
	Discover(ctx context.Context) (map[string][]types.Resource, error)
}
