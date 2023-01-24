package acmeprotocoldriver

import (
	"context"

	"github.com/upper-institute/ops-control/pkg/servicemesh"
	"go.uber.org/zap"
)

type certificateEnsurer struct {
	logger *zap.SugaredLogger
}

func NewCertificateEnsurer(
	logger *zap.SugaredLogger,
) servicemesh.EnvoyDiscoveryService {
	return &certificateEnsurer{
		logger,
	}
}

func (c *certificateEnsurer) Discover(ctx context.Context, resCh chan *servicemesh.Resources) {

}
