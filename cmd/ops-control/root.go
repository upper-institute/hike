package controllers

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	envoyctlr "github.com/upper-institute/ops-control/cmd/ops-control/envoy"
	"github.com/upper-institute/ops-control/cmd/ops-control/parameter"
	parameterctlr "github.com/upper-institute/ops-control/cmd/ops-control/parameter"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	"github.com/upper-institute/ops-control/internal/logger"
	otelgrpc "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"
)

const rootCmdUse = "ops-control"

var (
	cfgFile string

	serverListener net.Listener
	grpcServer     *grpc.Server
	serverMux      = http.NewServeMux()

	RootCmd = &cobra.Command{
		Use:   rootCmdUse,
		Short: "ops-control, functions to control cloud native operations",
		PersistentPostRunE: func(cmd *cobra.Command, args []string) error {

			opts := []grpc.ServerOption{
				grpc.StreamInterceptor(
					grpc_middleware.ChainStreamServer(
						otelgrpc.StreamServerInterceptor(),
						grpc_zap.StreamServerInterceptor(logger.Logger),
					),
				),
				grpc.UnaryInterceptor(
					grpc_middleware.ChainUnaryServer(
						otelgrpc.UnaryServerInterceptor(),
						grpc_zap.UnaryServerInterceptor(logger.Logger),
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

			if envoyctlr.RegisterServices(grpcServer, serverMux) {
				isGrpcServer = true
			}

			if isGrpcServer {
				serveGrpcServer()
			}

			logger.FlushLogger()

			return nil

		},
	}
)

func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {

	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/."+rootCmdUse+".yaml)")

	RootCmd.PersistentFlags().String("listenAddr", "0.0.0.0:7070", "Bind address to store gRPC server")
	RootCmd.PersistentFlags().Bool("tls", false, "Enable TLS protocol only on gRPC server")
	RootCmd.PersistentFlags().String("tlsKey", "", "PEM encoded private key file path")
	RootCmd.PersistentFlags().String("tlsCert", "", "PEM encoded certificate file path")
	RootCmd.PersistentFlags().Int("grpcMaxConcurrentStreams", 1000000, "Max concurrent streams for gRPC server")

	viper.BindPFlag("grpcServer.listenAddr", RootCmd.PersistentFlags().Lookup("listenAddr"))
	viper.BindPFlag("grpcServer.tls.enable", RootCmd.PersistentFlags().Lookup("tls"))
	viper.BindPFlag("grpcServer.tls.tlsKey", RootCmd.PersistentFlags().Lookup("tlsKey"))
	viper.BindPFlag("grpcServer.tls.tlsCert", RootCmd.PersistentFlags().Lookup("tlsCert"))
	viper.BindPFlag("grpcServer.grpc.maxConcurrentStreams", RootCmd.PersistentFlags().Lookup("grpcMaxConcurrentStreams"))

	RootCmd.AddCommand(envoyctlr.EnvoyCmd)
	RootCmd.AddCommand(parameterctlr.ParameterCmd)

	parameter.AttachParameterPullOptions(RootCmd.PersistentFlags())
	logger.AttachLoggingOptions(RootCmd.PersistentFlags(), viper.GetViper())

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

func serveGrpcServer() {

	reflection.Register(grpcServer)

	log.Println("Server listening at:", serverListener.Addr())

	server := &http.Server{Handler: &grpcMatcher{}}

	if err := server.Serve(serverListener); err != nil {
		log.Fatalln("Failed to serve because", err)
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
