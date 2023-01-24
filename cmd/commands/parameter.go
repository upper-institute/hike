package commands

import (
	"context"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/upper-institute/hike/internal"
)

var (
	parameterCmd = &cobra.Command{
		Use:   "parameter",
		Short: "Parameter management related commands",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			internal.LoadLogger(viper.GetViper())
			internal.LoadDrivers(context.Background())
		},
	}
)

func init() {

	parameterCmd.PersistentFlags().String("parameterUri", "", "Path to manipulate parameter")

	viper.BindPFlag("parameter.uri", parameterCmd.PersistentFlags().Lookup("parameterUri"))

	pullCmd.PersistentFlags().Bool("loadProcessEnvs", true, "Load envs from process")

	viper.BindPFlag("parameter.load.processEnvs", pullCmd.PersistentFlags().Lookup("loadProcessEnvs"))

	pullCmd.PersistentFlags().StringArray("saveFileFromKey", []string{}, "Save files only in the specified key")

	viper.BindPFlag("parameter.saveFileFromKey", pullCmd.PersistentFlags().Lookup("saveFileFromKey"))

	parameterCmd.AddCommand(pullCmd)

}
