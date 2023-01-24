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

	parameterCmd.PersistentFlags().String("parameter-uri", "", "Path to manipulate parameter")

	viper.BindPFlag("parameter.uri", parameterCmd.PersistentFlags().Lookup("parameter-uri"))

	pullCmd.PersistentFlags().Bool("load-process-envs", true, "Load envs from process")

	viper.BindPFlag("parameter.load.processEnvs", pullCmd.PersistentFlags().Lookup("load-process-envs"))

	pullCmd.PersistentFlags().StringArray("save-file-from-key", []string{}, "Save files only in the specified key")

	viper.BindPFlag("parameter.saveFileFromKey", pullCmd.PersistentFlags().Lookup("save-file-from-key"))

	parameterCmd.AddCommand(pullCmd)

}
