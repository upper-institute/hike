package controllers

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	envoyctlr "github.com/upper-institute/ops-control/cmd/controllers/envoy"
	parameterctlr "github.com/upper-institute/ops-control/cmd/controllers/parameter"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"
)

const rootCmdUse = "controllers"

var (
	cfgFile string

	listenAddr               string
	enableTls                bool
	tlsKey                   string
	tlsCert                  string
	grpcMaxConcurrentStreams int

	grpcServerListener net.Listener
	grpcServer         *grpc.Server

	isGrpcServer = false

	RootCmd = &cobra.Command{
		Use:   rootCmdUse,
		Short: "flipbook - Snapshot store",
		PersistentPostRunE: func(cmd *cobra.Command, args []string) error {

			if isGrpcServer {
				serveGrpcServer()
			}

			return nil

		},
		PersistentPreRun: func(cmd *cobra.Command, args []string) {

			opts := []grpc.ServerOption{}

			if enableTls {

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

			lis, err := net.Listen("tcp", listenAddr)
			if err != nil {
				log.Fatalln("failed to listen to store address", listenAddr, "because", err)
			}

			grpcServerListener = lis

			grpcServer = grpc.NewServer(opts...)

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
	cobra.OnInitialize(initConfig)

	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/."+rootCmdUse+".yaml)")

	RootCmd.PersistentFlags().StringVar(&listenAddr, "listenAddr", "0.0.0.0:7070", "Bind address to store gRPC server")
	RootCmd.PersistentFlags().BoolVar(&enableTls, "tls", false, "Enable TLS protocol only on gRPC server")
	RootCmd.PersistentFlags().StringVar(&tlsKey, "tlsKey", "", "PEM encoded private key file path")
	RootCmd.PersistentFlags().StringVar(&tlsCert, "tlsCert", "", "PEM encoded certificate file path")
	RootCmd.PersistentFlags().IntVar(&grpcMaxConcurrentStreams, "grpcMaxConcurrentStreams", 1000000, "Max concurrent streams for gRPC server")

	viper.BindPFlag("grpcServer.listenAddr", RootCmd.Flags().Lookup("listenAddr"))
	viper.BindPFlag("grpcServer.tls.enable", RootCmd.Flags().Lookup("tls"))
	viper.BindPFlag("grpcServer.tls.tlsKey", RootCmd.Flags().Lookup("tlsKey"))
	viper.BindPFlag("grpcServer.tls.tlsCert", RootCmd.Flags().Lookup("tlsCert"))
	viper.BindPFlag("grpcServer.grpc.maxConcurrentStreams", RootCmd.Flags().Lookup("grpcMaxConcurrentStreams"))

	RootCmd.AddCommand(envoyctlr.EnvoyCmd)
	RootCmd.AddCommand(parameterctlr.ParameterCmd)

}

func serveGrpcServer() {

	if envoyctlr.RegisterServices(grpcServer) {
		isGrpcServer = true
	}

	reflection.Register(grpcServer)

	log.Println("Server listening at:", grpcServerListener.Addr())

	if err := grpcServer.Serve(grpcServerListener); err != nil {
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

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
