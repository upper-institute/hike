package envoy

import (
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/upper-institute/ops-control/internal/logger"
	"github.com/upper-institute/ops-control/providers/envoy"
	"google.golang.org/grpc"
)

var (
	EnvoyCmd = &cobra.Command{
		Use:   "envoy",
		Short: "Envoy related controls",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			logger.LoadLogger(viper.GetViper())
		},
	}
)

func init() {

	EnvoyCmd.PersistentFlags().Duration("discoveryMinInterval", 30*time.Second, "Discovery minimum interval to reload")

	viper.BindPFlag("envoy.discoveryMinInterval", EnvoyCmd.PersistentFlags().Lookup("discoveryMinInterval"))

	xdsServerCmd.PersistentFlags().String("nodeId", "ops-control-node", "Tell envoy which node id to use")
	xdsServerCmd.PersistentFlags().String("xdsClusterName", "xds-cluster", "Pointer to service discovery for resources")
	xdsServerCmd.PersistentFlags().String("parameterPathTag", "parameter_path", "Tag in the resource to discover parameter envs and files")

	viper.BindPFlag("envoy.nodeId", xdsServerCmd.PersistentFlags().Lookup("nodeId"))
	viper.BindPFlag("envoy.xdsCluster.name", xdsServerCmd.PersistentFlags().Lookup("xdsClusterName"))
	viper.BindPFlag("envoy.parameter.pathTag", xdsServerCmd.PersistentFlags().Lookup("parameterPathTag"))

	xdsServerCmd.PersistentFlags().Bool("enableAwsCloudMap", false, "Enable AWS Cloud Map service discovery")
	xdsServerCmd.PersistentFlags().StringSlice("awsCloudMapNamespaces", []string{}, "AWS CloudMap (Service Discovery) namespaces to watch for services and instances")
	xdsServerCmd.PersistentFlags().Bool("enableAwsRoute53", false, "AWS Route53 domain registry for service discovery (the zone must match")

	viper.BindPFlag("envoy.aws.cloudMap", xdsServerCmd.PersistentFlags().Lookup("enableAwsCloudMap"))
	viper.BindPFlag("envoy.aws.cloudMap.namespaces", xdsServerCmd.PersistentFlags().Lookup("awsCloudMapNamespaces"))
	viper.BindPFlag("envoy.aws.route53", xdsServerCmd.PersistentFlags().Lookup("enableAwsRoute53"))

	EnvoyCmd.AddCommand(xdsServerCmd)

}

func RegisterServices(grpcServer *grpc.Server) bool {

	log := logger.SugaredLogger

	if xdsServer != nil {
		log.Info("Registering XDS Services to gRPC Server")
		envoy.RegisterXDSServices(grpcServer, xdsServer)
		return true
	}

	return false

}
