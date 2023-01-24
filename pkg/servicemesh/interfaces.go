package servicemesh

import (
	"context"

	sdapi "github.com/upper-institute/hike/proto/api/service-discovery"
)

type EnvoyDiscoveryService interface {
	Discover(ctx context.Context, svcCh chan *sdapi.Service)
}
