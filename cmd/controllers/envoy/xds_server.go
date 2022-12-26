package envoy

import (
	"context"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	awsctlr "github.com/upper-institute/ops-control/internal/aws"
	envoyctlr "github.com/upper-institute/ops-control/internal/envoy"

	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	serverv3 "github.com/envoyproxy/go-control-plane/pkg/server/v3"
)

var (
	xdsServer serverv3.Server

	xdsServerCmd = &cobra.Command{
		Use:   "xds-server",
		Short: "Run xDS server",
		RunE: func(cmd *cobra.Command, args []string) error {

			cache := cache.NewSnapshotCache(false, cache.IDHash{}, nil)

			go func() {

				var (
					awsFrontProxy *awsctlr.FrontProxy

					genericConfiguration = &envoyctlr.GenericConfiguration{}

					discoveryMinInterval = viper.GetDuration("envoy.discoveryMinInterval")
				)

				for {

					log.Println("Starting new discovery execution")

					ctx := context.Background()

					genericConfiguration.Reset()

					if viper.GetBool("envoy.enableAwsEnvoyFrontProxy") {

						log.Println("AWS front proxy discovery enabled")

						if awsFrontProxy == nil {

							config, err := config.LoadDefaultConfig(ctx)
							if err != nil {
								log.Fatalln(err.Error())
							}

							awsFrontProxy = &awsctlr.FrontProxy{
								Config:               config,
								NamespacesNames:      viper.GetStringSlice("envoy.aws.cloudMap.namespaces"),
								GenericConfiguration: genericConfiguration,
							}

						}

						err := awsFrontProxy.LoadConfigurationFromCloudMap(ctx)
						if err != nil {
							log.Fatalln(err.Error())
						}

					}

					snapshot, err := genericConfiguration.DoSnapshotCache()
					if err != nil {
						log.Fatalln(err.Error())
					}

					err = cache.SetSnapshot(ctx, viper.GetString("envoy.nodeId"), snapshot)
					if err != nil {
						log.Fatalln(err.Error())
					}

					log.Println("Service discovery execution ended successfully")

					time.Sleep(discoveryMinInterval)
				}
			}()

			ctx := context.Background()

			xdsServer = serverv3.NewServer(ctx, cache, nil)

			return nil
		},
	}
)
