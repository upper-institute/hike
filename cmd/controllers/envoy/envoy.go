package envoy

import (
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/upper-institute/ops-control/internal/envoy"
	"google.golang.org/grpc"
)

var (
	EnvoyCmd = &cobra.Command{
		Use:   "envoy",
		Short: "Envoy related controls",
	}
)

func init() {

	EnvoyCmd.AddCommand(xdsServerCmd)

	EnvoyCmd.PersistentFlags().Duration("discoveryMinInterval", 5*time.Second, "Discovery minimum interval to reload")
	EnvoyCmd.PersistentFlags().Bool("enableAwsEnvoyFrontProxy", false, "Enable AWS envoy front-proxy")

	viper.BindPFlag("envoy.discoveryMinInterval", EnvoyCmd.PersistentFlags().Lookup("discoveryMinInterval"))
	viper.BindPFlag("envoy.enableAwsEnvoyFrontProxy", EnvoyCmd.PersistentFlags().Lookup("enableAwsEnvoyFrontProxy"))

	xdsServerCmd.PersistentFlags().String("nodeId", "default-node", "Tell envoy which node id to use")
	xdsServerCmd.PersistentFlags().StringSlice("awsCloudMapNamespaces", []string{}, "AWS CloudMap (Service Discovery) namespaces to watch for services and instances")

	viper.BindPFlag("envoy.nodeId", xdsServerCmd.PersistentFlags().Lookup("nodeId"))
	viper.BindPFlag("envoy.aws.cloudMap.namespaces", xdsServerCmd.PersistentFlags().Lookup("awsCloudMapNamespaces"))

}

func RegisterServices(grpcServer *grpc.Server) bool {

	if xdsServer != nil {
		envoy.RegisterXDSServices(grpcServer, xdsServer)
		return true
	}

	return false

}
