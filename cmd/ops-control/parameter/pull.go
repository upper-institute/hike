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
	paramapi "github.com/upper-institute/ops-control/gen/api/parameter"
	"github.com/upper-institute/ops-control/internal/logger"
	"github.com/upper-institute/ops-control/internal/parameter"
	"github.com/upper-institute/ops-control/providers/aws"
)

func LoadParameterStorage(ctx context.Context) (parameter.Storage, error) {

	var (
		log       = logger.SugaredLogger
		paramFile parameter.Storage
	)

	switch {
	case viper.GetBool("parameter.load.aws.s3ParameterStorage"):

		log.Infow("Configuring AWS S3 parameter storage")

		config, err := config.LoadDefaultConfig(ctx)
		if err != nil {
			log.Error(err)
			return nil, err
		}

		s3Client := s3.NewFromConfig(config)

		paramFile = aws.NewS3ParameterStorage(s3Client, logger.SugaredLogger)
	default:

		log.Warnln("No parameter file configured")

	}

	return paramFile, nil

}

func LoadParameterStore(ctx context.Context) (parameter.Store, error) {

	var (
		log            = logger.SugaredLogger
		parameterStore parameter.Store
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
	default:

		log.Warnln("No parameter store configured")

	}

	return parameterStore, nil

}

func LoadParameterCacheOptions(ctx context.Context) (*parameter.CacheOptions, error) {

	parameterStorage, err := LoadParameterStorage(ctx)
	if err != nil {
		return nil, err
	}

	parameterStore, err := LoadParameterStore(ctx)
	if err != nil {
		return nil, err
	}

	return &parameter.CacheOptions{
		Store: parameterStore,
		ParameterOptions: &parameter.ParameterOptions{
			Uploader:   parameterStorage,
			Downloader: parameterStorage,
			Writer:     parameterStore,
			Logger:     logger.SugaredLogger,
		},
	}, nil

}

var (
	pullCmd = &cobra.Command{
		Use:   "pull",
		Short: "Produce output from template file, the first argument is optional, if provided will be dumped as a KEY=VALUE for envs loaded",
		Args:  cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) error {

			var (
				log             = logger.SugaredLogger
				parameterUri    = viper.GetString("parameter.uri")
				loadProcessEnvs = viper.GetBool("parameter.load.processEnvs")
			)

			ctx := context.Background()

			paramCacheOpts, err := LoadParameterCacheOptions(ctx)
			if err != nil {
				return err
			}

			paramCache, err := paramCacheOpts.NewFromURLString(parameterUri)
			if err != nil {
				return err

			}

			if loadProcessEnvs {

				err = paramCache.RestoreFromProcessEnvs()
				if err != nil {
					return err
				}

			}

			err = paramCache.Restore(ctx)
			if err != nil {
				return err
			}

			saveFileFromKey := viper.GetStringSlice("parameter.saveFileFromKey")

			if len(saveFileFromKey) > 0 {

				for _, fileKey := range saveFileFromKey {

					log.Infow("Saving file from key", "file_key", fileKey)

					if !paramCache.Has(fileKey) {
						return fmt.Errorf("File not found by key: %s", fileKey)
					}

					param := paramCache.Get(fileKey)

					err = param.Load(ctx)
					if err != nil {
						return fmt.Errorf("Unable to load file from parameter: %s", saveFileFromKey)
					}

					destination := param.GetFragment()

					err = os.WriteFile(destination, param.GetFile().Bytes(), 0644)
					if err != nil {
						return err
					}

				}

			}

			if len(args) == 1 {

				log.Infow("Writing env export file", "target", args[0])

				envFile, err := os.Create(args[0])
				if err != nil {
					return err
				}

				params := paramCache.List()

				for _, param := range params {

					if param.GetType() != paramapi.ParameterType_PARAMETER_TYPE_VAR {
						continue
					}

					envFile.WriteString(param.GetKey())
					envFile.WriteString("=")
					envFile.WriteString(param.GetFragment())
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
