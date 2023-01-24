package drivers

import (
	"context"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	acmeprotocoldriver "github.com/upper-institute/ops-control/pkg/drivers/acmeprotocol"
	"github.com/upper-institute/ops-control/pkg/helpers"
	"github.com/upper-institute/ops-control/pkg/servicemesh"
	"go.uber.org/zap"
)

type ACMEProtocolDriver struct {
	Logger *zap.SugaredLogger
}

func (d *ACMEProtocolDriver) Bind(cfg *viper.Viper, flagSet *pflag.FlagSet) {

	binder := &helpers.FlagBinder{cfg, flagSet}

}

func (d *ACMEProtocolDriver) Load(ctx context.Context) error {

	return nil

}

func (d *ACMEProtocolDriver) NewCertificateEnsurer() servicemesh.EnvoyDiscoveryService {
	return acmeprotocoldriver.NewCertificateEnsurer(d.Logger)
}
