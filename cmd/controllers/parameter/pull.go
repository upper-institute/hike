package parameter

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/upper-institute/ops-control/internal/logger"
	"github.com/upper-institute/ops-control/internal/parameter"
	"github.com/upper-institute/ops-control/providers/aws"
)

func LoadParameterFileDownloader(ctx context.Context) (parameter.ParameterFileDownloader, error) {

	var (
		log        = logger.SugaredLogger
		downloader parameter.ParameterFileDownloader
	)

	switch {
	case viper.GetBool("parameter.load.aws.s3ParameterFileDownloader"):

		log.Infow("Configuring AWS S3 parameter file downloader")

		config, err := config.LoadDefaultConfig(ctx)
		if err != nil {
			log.Error(err)
			return nil, err
		}

		s3Client := s3.NewFromConfig(config)

		downloader = aws.NewS3ParameterFileDownloader(s3Client, logger.SugaredLogger)

	}

	return downloader, nil

}

func LoadParameterStore(ctx context.Context) (parameter.ParameterStore, error) {

	var (
		log            = logger.SugaredLogger
		parameterStore parameter.ParameterStore
	)

	switch {
	case viper.GetBool("parameter.load.aws.ssmParameterStore"):

		log.Infow("Configuring AWS SSM parameter store")

		config, err := config.LoadDefaultConfig(ctx)
		if err != nil {
			log.Error(err)
			return nil, err
		}

		ssmClient := ssm.NewFromConfig(config)

		parameterStore = aws.NewSSMParameterStore(ssmClient, logger.SugaredLogger)

	}

	return parameterStore, nil

}

func LoadParameterProviders(ctx context.Context) (parameter.ParameterStore, parameter.ParameterFileDownloader, error) {

	downloader, err := LoadParameterFileDownloader(ctx)
	if err != nil {
		return nil, nil, err
	}

	parameterStore, err := LoadParameterStore(ctx)
	if err != nil {
		return nil, nil, err
	}

	return parameterStore, downloader, nil

}

type paramPath string

func (p paramPath) GetParameterPath() string {
	return string(p)
}

var (
	pullCmd = &cobra.Command{
		Use:   "pull",
		Short: "Produce output from template file, the first argument is optional, if provided will be dumped as a KEY=VALUE for envs loaded",
		Args:  cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) error {

			var (
				log             = logger.SugaredLogger
				parameterPath   = paramPath(viper.GetString("parameter.path"))
				loadProcessEnvs = viper.GetBool("parameter.load.processEnvs")

				err            error
				downloader     parameter.ParameterFileDownloader
				parameterStore parameter.ParameterStore
			)

			ctx := context.Background()

			downloader, err = LoadParameterFileDownloader(ctx)
			if err != nil {
				return err
			}

			parameterStore, err = LoadParameterStore(ctx)
			if err != nil {
				return err
			}

			parameterSet := parameter.NewParameterSet(downloader, logger.SugaredLogger)

			err = parameterStore.Load(ctx, parameterPath, parameterSet)
			if err != nil {
				return err
			}

			if loadProcessEnvs {

				err = parameter.LoadParametersFromProcessEnv(parameterSet)
				if err != nil {
					return err
				}

			}

			if viper.GetBool("parameter.saveAllFiles") {

				err = parameterSet.SaveAllFiles(ctx)
				if err != nil {
					return err
				}

			}

			saveFileFromKey := viper.GetString("parameter.saveFileFromKey")

			if len(saveFileFromKey) > 0 {

				log.Infow("Saving file from key", "save_file_from_key", saveFileFromKey)

				if !parameterSet.HasFile(saveFileFromKey) {
					return fmt.Errorf("File not found by key: %s", saveFileFromKey)
				}

				err = parameterSet.SaveFile(ctx, saveFileFromKey)
				if err != nil {
					return err
				}

			}

			if len(args) == 1 {

				log.Infow("Writing env export file", "target", args[0])

				envFile, err := os.Create(args[0])
				if err != nil {
					return err
				}

				allEnvs := parameterSet.GetAllEnvs()

				for key, value := range allEnvs {
					envFile.WriteString(key)
					envFile.WriteString("=")
					envFile.WriteString(value)
				}

				err = envFile.Close()
				if err != nil {
					return err
				}

			}

			return nil
		},
	}
)
