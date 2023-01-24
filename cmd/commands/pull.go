package commands

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	internal "github.com/upper-institute/hike/internal"
	paramapi "github.com/upper-institute/hike/proto/api/parameter"
)

var (
	pullCmd = &cobra.Command{
		Use:   "pull",
		Short: "Produce output from template file, the first argument is optional, if provided will be dumped as a KEY=VALUE for envs loaded",
		Args:  cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) error {

			var (
				log             = internal.SugaredLogger
				parameterUri    = viper.GetString("parameter.uri")
				loadProcessEnvs = viper.GetBool("parameter.load.processEnvs")
			)

			ctx := context.Background()

			paramCache, err := internal.ParameterSourceOptions.NewFromURLString(parameterUri)
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

					if param.GetType() != paramapi.ParameterType_PT_VAR {
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
