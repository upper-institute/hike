package domainregistry

import (
	"context"

	service_discovery "github.com/upper-institute/ops-control/gen/api/service-discovery"
)

type DomainRegistryService interface {
	RegisterIngressDomain(ctx context.Context, ingressDomain *service_discovery.IngressDomain) error
}
