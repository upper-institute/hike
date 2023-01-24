package commands

import (
	"context"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/upper-institute/hike/internal"
)

var (
	envoyCmd = &cobra.Command{
		Use:   "envoy",
		Short: "Envoy related controls",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			internal.LoadLogger(viper.GetViper())
			internal.LoadDrivers(context.Background())
		},
	}
)

func init() {

	envoyCmd.PersistentFlags().Duration("discoveryMinInterval", 30*time.Second, "Discovery minimum interval to reload")
	viper.BindPFlag("envoy.discoveryMinInterval", envoyCmd.PersistentFlags().Lookup("discoveryMinInterval"))

	xdsServerCmd.PersistentFlags().String("nodeId", "hike-node", "Tell envoy which node id to use")
	viper.BindPFlag("envoy.nodeId", xdsServerCmd.PersistentFlags().Lookup("nodeId"))

	envoyCmd.AddCommand(xdsServerCmd)

}
