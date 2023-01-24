package servicemesh

import (
	"context"

	service_discovery "github.com/upper-institute/ops-control/gen/api/service-discovery"
)

type EnvoyDiscoveryService interface {
	Discover(ctx context.Context, svcCh chan *service_discovery.Service)
}
