package envoy

import (
	"context"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/spf13/cobra"
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
				)

				for {

					ctx := context.Background()

					if enableAwsEnvoyFrontProxy {

						if awsFrontProxy == nil {

							config, err := config.LoadDefaultConfig(ctx)
							if err != nil {
								log.Fatalln(err.Error())
							}

							awsFrontProxy = &awsctlr.FrontProxy{
								Config:               config,
								NamespacesNames:      awsCloudMapNamespaces,
								GenericConfiguration: genericConfiguration,
							}

						}

						err := awsFrontProxy.LoadConfigurationFromCloudMap(ctx)
						if err != nil {
							log.Fatalln(err.Error())
						}

					}

					genericConfiguration.IncrementVersion()

					snapshot, err := genericConfiguration.DoSnapshotCache()
					if err != nil {
						log.Fatalln(err.Error())
					}

					err = cache.SetSnapshot(ctx, nodeId, snapshot)
					if err != nil {
						log.Fatalln(err.Error())
					}

					time.Sleep(discoveryMinInterval)
				}
			}()

			ctx := context.Background()

			xdsServer = serverv3.NewServer(ctx, cache, nil)

			return nil
		},
	}
)
