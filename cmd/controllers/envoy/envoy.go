package envoy

import (
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/upper-institute/ops-control/internal/envoy"
	"google.golang.org/grpc"
)

var (
	discoveryMinInterval     time.Duration
	enableAwsEnvoyFrontProxy bool
	nodeId                   string

	EnvoyCmd = &cobra.Command{
		Use:   "envoy",
		Short: "Envoy related controls",
	}
)

func init() {

	EnvoyCmd.AddCommand(xdsServerCmd)

	EnvoyCmd.PersistentFlags().DurationVar(&discoveryMinInterval, "discoveryMinInterval", 5*time.Second, "Discovery minimum interval to reload")
	EnvoyCmd.PersistentFlags().BoolVar(&enableAwsEnvoyFrontProxy, "enableAwsEnvoyFrontProxy", false, "Enable AWS envoy front-proxy")

	viper.BindPFlag("envoy.discoveryMinInterval", EnvoyCmd.Flags().Lookup("discoveryMinInterval"))
	viper.BindPFlag("envoy.enableAwsFrontProxy", EnvoyCmd.Flags().Lookup("enableAwsEnvoyFrontProxy"))

	xdsServerCmd.PersistentFlags().StringVar(&nodeId, "nodeId", "default-node", "Tell envoy which node id to use")

	viper.BindPFlag("envoy.nodeId", EnvoyCmd.Flags().Lookup("nodeId"))

}

func RegisterServices(grpcServer *grpc.Server) bool {

	if xdsServer != nil {
		envoy.RegisterXDSServices(grpcServer, xdsServer)
		return true
	}

	return false

}
