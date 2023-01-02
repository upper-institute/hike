package parameter

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/upper-institute/ops-control/internal/logger"
)

var (
	ParameterCmd = &cobra.Command{
		Use:   "parameter",
		Short: "Parameter management related commands",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			logger.LoadLogger(viper.GetViper())
		},
	}
)

func AttachParameterPullOptions(flagSet *pflag.FlagSet) {

	flagSet.Bool("loadAwsSsmParameterStore", false, "Use AWS SSM Parameter Store to pull parameters (files and envs)")
	flagSet.Bool("loadAwsS3ParameterFileDownloader", false, "Use AWS S3 Parameter File Downloader to download files from parameter store")

	viper.BindPFlag("parameter.load.aws.ssmParameterStore", flagSet.Lookup("loadAwsSsmParameterStore"))
	viper.BindPFlag("parameter.load.aws.s3ParameterFileDownloader", flagSet.Lookup("loadAwsS3ParameterFileDownloader"))

}

func init() {

	ParameterCmd.PersistentFlags().String("parameterPath", "", "Path to manipulate parameter")

	viper.BindPFlag("parameter.path", ParameterCmd.PersistentFlags().Lookup("parameterPath"))

	pullCmd.PersistentFlags().Bool("loadProcessEnvs", true, "Load envs from process")

	viper.BindPFlag("parameter.load.processEnvs", pullCmd.PersistentFlags().Lookup("loadProcessEnvs"))

	pullCmd.PersistentFlags().Bool("saveAllFiles", false, "Save all files from parameter store")
	pullCmd.PersistentFlags().String("saveFileFromKey", "", "Save files only in the specified key")

	viper.BindPFlag("parameter.saveAllFiles", pullCmd.PersistentFlags().Lookup("saveAllFiles"))
	viper.BindPFlag("parameter.saveFileFromKey", pullCmd.PersistentFlags().Lookup("saveFileFromKey"))

	ParameterCmd.AddCommand(pullCmd)

}
