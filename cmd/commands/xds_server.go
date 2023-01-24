package commands

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/upper-institute/hike/pkg/servicemesh"
)

var (
	discoveryOptions *servicemesh.EnvoyDiscoveryOptions

	xdsServerCmd = &cobra.Command{
		Use:   "xds-server",
		Short: "Run xDS server",
		RunE: func(cmd *cobra.Command, args []string) error {

			var (
				discoveryMinInterval = viper.GetDuration("envoy.discoveryMinInterval")
				nodeId               = viper.GetString("envoy.nodeId")
			)

			discoveryOptions = &servicemesh.EnvoyDiscoveryOptions{
				NodeID:        nodeId,
				WatchInterval: discoveryMinInterval,
			}

			return nil
		},
	}
)
