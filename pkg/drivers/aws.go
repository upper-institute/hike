package drivers

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/servicediscovery"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	awsdriver "github.com/upper-institute/ops-control/pkg/drivers/aws"
	"github.com/upper-institute/ops-control/pkg/helpers"
	"github.com/upper-institute/ops-control/pkg/parameter"
	"github.com/upper-institute/ops-control/pkg/servicemesh"
	"go.uber.org/zap"
)

const (
	DriversAwsSsmParameterStoreEnable        = "drivers.aws.ssm.parameter.store.enable"
	DriversAwsS3ParameterStorageEnable       = "drivers.aws.s3.parameter.storage.enable"
	DriversAwsRoute53DomainRegistryEnable    = "drivers.aws.route53.domain.registry.enable"
	DriversAwsCloudMapServiceDiscoveryEnable = "drivers.aws.cloudmap.service.discovery.enable"
	DriversAwsCloudMapNamespacesNames        = "drivers.aws.cloudmap.namespaces.names"
	DriversAwsCloudMapParameterUriTag        = "drivers.aws.cloudmap.parameter.uri.tag"
)

type AWSDriver struct {
	Logger *zap.SugaredLogger

	config aws.Config

	binder *helpers.FlagBinder
}

func (d *AWSDriver) Bind(cfg *viper.Viper, flagSet *pflag.FlagSet) {

	d.binder = &helpers.FlagBinder{cfg, flagSet}

	d.binder.BindBool(DriversAwsSsmParameterStoreEnable, false, "Use AWS SSM Parameter Store to pull/push parameters (files and envs)")
	d.binder.BindBool(DriversAwsS3ParameterStorageEnable, false, "Use AWS S3 Parameter Storage to download/uploade files from parameter store")
	d.binder.BindBool(DriversAwsRoute53DomainRegistryEnable, false, "Use AWS Route53 domain registry service")
	d.binder.BindBool(DriversAwsCloudMapServiceDiscoveryEnable, false, "Use AWS Cloud Map service discovery service")
	d.binder.BindStringSlice(DriversAwsCloudMapNamespacesNames, []string{}, "AWS CloudMap (Service Discovery) namespaces to watch for services and instances")
	d.binder.BindString(DriversAwsCloudMapParameterUriTag, "parameter_uri", "Tag in the Cloud Map Service resource to discover parameter envs and files")

}

func (d *AWSDriver) Load(ctx context.Context) error {

	config, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return err
	}

	d.config = config

	return nil

}

func (d *AWSDriver) NewSSMParameterStore() parameter.Store {

	ssmClient := ssm.NewFromConfig(d.config)

	return awsdriver.NewSSMParameterStore(ssmClient, d.Logger)

}

func (d *AWSDriver) NewS3ParameterStorage() parameter.Storage {

	s3Client := s3.NewFromConfig(d.config)

	return awsdriver.NewS3ParameterStorage(s3Client, d.Logger)

}

func (d *AWSDriver) NewCloudMapServiceDiscovery() servicemesh.EnvoyDiscoveryService {

	cloudMapClient := servicediscovery.NewFromConfig(d.config)

	return awsdriver.NewCloudMapServiceDiscovery(
		d.binder.Viper.GetStringSlice(DriversAwsCloudMapNamespacesNames),
		d.binder.Viper.GetString(DriversAwsCloudMapParameterUriTag),
		cloudMapClient,
		d.Logger,
	)

}

func (d *AWSDriver) NewRoute53DomainRegistry() servicemesh.EnvoyDiscoveryService {

	route53Client := route53.NewFromConfig(d.config)

	return awsdriver.NewRoute53DomainRegistry(route53Client, d.Logger)

}
