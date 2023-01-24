package internal

import (
	"context"

	"github.com/upper-institute/hike/pkg/drivers"
	"github.com/upper-institute/hike/pkg/parameter"
	"github.com/upper-institute/hike/pkg/servicemesh"
	"go.uber.org/zap"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var (
	AWSDriver = &drivers.AWSDriver{}

	ParameterSourceOptions = &parameter.SourceOptions{
		ParameterOptions: &parameter.ParameterOptions{},
	}

	Drivers = []Driver{
		AWSDriver,
	}

	EnvoyDiscoveryServices = []servicemesh.EnvoyDiscoveryService{}
)

func AttachDriversOptions(flagSet *pflag.FlagSet, cfg *viper.Viper) {

	for _, driver := range Drivers {
		driver.Bind(flagSet, cfg)
	}

}

func LoadDrivers(ctx context.Context) {

	for _, driver := range Drivers {

		err := driver.Load(ctx, SugaredLogger)

		if err != nil {
			SugaredLogger.Fatalw("Error loading driver", "error", err)
		}

		driver.ApplyParameterSourceOptions(ParameterSourceOptions)

	}

	for _, driver := range Drivers {
		EnvoyDiscoveryServices = append(
			EnvoyDiscoveryServices,
			driver.GetEnvoyDiscoveryServices(ParameterSourceOptions)...,
		)
	}

}

type Driver interface {
	ApplyParameterSourceOptions(opts *parameter.SourceOptions)

	GetEnvoyDiscoveryServices(opts *parameter.SourceOptions) []servicemesh.EnvoyDiscoveryService

	Bind(flagSet *pflag.FlagSet, cfg *viper.Viper)
	Load(ctx context.Context, logger *zap.SugaredLogger) error
}
