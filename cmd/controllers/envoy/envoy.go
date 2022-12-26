package envoy

import (
	"time"

	"github.com/spf13/cobra"
	"github.com/upper-institute/ops-control/internal/envoy"
	"google.golang.org/grpc"
)

var (
	discoveryMinInterval     time.Duration
	enableAwsEnvoyFrontProxy bool
	nodeId                   string
	awsCloudMapNamespaces    []string

	EnvoyCmd = &cobra.Command{
		Use:   "envoy",
		Short: "Envoy related controls",
	}
)

func init() {

	EnvoyCmd.AddCommand(xdsServerCmd)

	EnvoyCmd.PersistentFlags().DurationVar(&discoveryMinInterval, "discoveryMinInterval", 5*time.Second, "Discovery minimum interval to reload")
	EnvoyCmd.PersistentFlags().BoolVar(&enableAwsEnvoyFrontProxy, "enableAwsEnvoyFrontProxy", false, "Enable AWS envoy front-proxy")

	xdsServerCmd.PersistentFlags().StringVar(&nodeId, "nodeId", "default-node", "Tell envoy which node id to use")
	xdsServerCmd.PersistentFlags().StringSliceVar(&awsCloudMapNamespaces, "awsCloudMapNamespaces", []string{}, "AWS CloudMap (Service Discovery) namespaces to watch for services and instances")

}

func RegisterServices(grpcServer *grpc.Server) bool {

	if xdsServer != nil {
		envoy.RegisterXDSServices(grpcServer, xdsServer)
		return true
	}

	return false

}
