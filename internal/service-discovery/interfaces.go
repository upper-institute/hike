package servicediscovery

import (
	"context"

	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	domainregistry "github.com/upper-institute/ops-control/internal/domain-registry"
	"github.com/upper-institute/ops-control/internal/parameter"
)

type ServiceDiscoveryService interface {
	SetParameterStore(parameterStore parameter.ParameterStore)
	SetParameterFileDownloader(parameterFileDownloader parameter.ParameterFileDownloader)
	SetDomainRegistry(domainRegistry domainregistry.DomainRegistryService)
	Discover(ctx context.Context) (map[string][]types.Resource, error)
}
