package commands

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/upper-institute/hike/internal"
	otelgrpc "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"
)

const rootCmdUse = "hike"

var (
	cfgFile string

	serverListener net.Listener
	grpcServer     *grpc.Server
	serverMux      = http.NewServeMux()

	rootCmd = &cobra.Command{
		Use:   rootCmdUse,
		Short: "hike, functions to control cloud native operations",
		PersistentPostRunE: func(cmd *cobra.Command, args []string) error {

			log := internal.SugaredLogger

			opts := []grpc.ServerOption{
				grpc.StreamInterceptor(
					grpc_middleware.ChainStreamServer(
						otelgrpc.StreamServerInterceptor(),
						grpc_zap.StreamServerInterceptor(internal.Logger),
					),
				),
				grpc.UnaryInterceptor(
					grpc_middleware.ChainUnaryServer(
						otelgrpc.UnaryServerInterceptor(),
						grpc_zap.UnaryServerInterceptor(internal.Logger),
					),
				),
			}

			if viper.GetBool("grpcServer.enableTls") {

				tlsCert := viper.GetString("grpcServer.tls.cert")
				tlsKey := viper.GetString("grpcServer.tls.key")

				cert, err := tls.LoadX509KeyPair(tlsCert, tlsKey)
				if err != nil {
					log.Fatalln("failed load TLS certificate (", tlsCert, ") or key (", tlsKey, ") because", err)
				}

				config := &tls.Config{
					Certificates: []tls.Certificate{cert},
					ClientAuth:   tls.VerifyClientCertIfGiven,
				}

				opts = append(opts, grpc.Creds(credentials.NewTLS(config)))

			}

			listenAddr := viper.GetString("grpcServer.listenAddr")

			lis, err := net.Listen("tcp", listenAddr)
			if err != nil {
				log.Fatalln("failed to listen to store address", listenAddr, "because", err)
			}

			serverListener = lis

			grpcServer = grpc.NewServer(opts...)

			isGrpcServer := false

			if discoveryOptions != nil {

				isGrpcServer = true

				discoveryOptions.Services = internal.EnvoyDiscoveryServices

				discoveryServer, err := discoveryOptions.NewServer(context.Background(), internal.SugaredLogger)

				if err != nil {
					log.Fatalln(err)
				}

				discoveryServer.StartDiscoveryCycle()

				discoveryServer.Register(grpcServer)

			}

			if isGrpcServer {

				reflection.Register(grpcServer)

				log.Infow("Server listening", "address", serverListener.Addr())

				server := &http.Server{Handler: &grpcMatcher{}}

				if err := server.Serve(serverListener); err != nil {
					log.Fatalln("Failed to serve because", err)
				}

			}

			internal.FlushLogger()

			return nil

		},
	}
)

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/."+rootCmdUse+".yaml)")

	rootCmd.PersistentFlags().String("listen-addr", "0.0.0.0:7070", "Bind address to store gRPC server")
	rootCmd.PersistentFlags().Bool("tls", false, "Enable TLS protocol only on gRPC server")
	rootCmd.PersistentFlags().String("tls-key", "", "PEM encoded private key file path")
	rootCmd.PersistentFlags().String("tls-cert", "", "PEM encoded certificate file path")
	rootCmd.PersistentFlags().Int("grpc-max-concurrent-streams", 1000000, "Max concurrent streams for gRPC server")

	viper.BindPFlag("grpcServer.listenAddr", rootCmd.PersistentFlags().Lookup("listen-addr"))
	viper.BindPFlag("grpcServer.tls.enable", rootCmd.PersistentFlags().Lookup("tls"))
	viper.BindPFlag("grpcServer.tls.tlsKey", rootCmd.PersistentFlags().Lookup("tls-key"))
	viper.BindPFlag("grpcServer.tls.tlsCert", rootCmd.PersistentFlags().Lookup("tls-cert"))
	viper.BindPFlag("grpcServer.grpc.maxConcurrentStreams", rootCmd.PersistentFlags().Lookup("grpc-max-concurrent-streams"))

	internal.AttachLoggingOptions(rootCmd.PersistentFlags(), viper.GetViper())
	internal.AttachDriversOptions(rootCmd.PersistentFlags(), viper.GetViper())

	rootCmd.AddCommand(envoyCmd)
	rootCmd.AddCommand(parameterCmd)

	cobra.OnInitialize(initConfig)

}

type grpcMatcher struct{}

func (g *grpcMatcher) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	ct := r.Header.Get("Content-Type")
	if r.ProtoMajor == 2 && strings.Contains(ct, "application/grpc") {
		grpcServer.ServeHTTP(w, r)
	} else {
		serverMux.ServeHTTP(w, r)
	}

}

func initConfig() {

	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name "." + rootCmdUse (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName("." + rootCmdUse)
	}

	viper.SetEnvPrefix("")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}

}
