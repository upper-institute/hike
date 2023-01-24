package servicemesh

import (
	"bytes"
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"

	clusterservice "github.com/envoyproxy/go-control-plane/envoy/service/cluster/v3"
	discoverygrpc "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	endpointservice "github.com/envoyproxy/go-control-plane/envoy/service/endpoint/v3"
	listenerservice "github.com/envoyproxy/go-control-plane/envoy/service/listener/v3"
	routeservice "github.com/envoyproxy/go-control-plane/envoy/service/route/v3"
	runtimeservice "github.com/envoyproxy/go-control-plane/envoy/service/runtime/v3"
	secretservice "github.com/envoyproxy/go-control-plane/envoy/service/secret/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	serverv3 "github.com/envoyproxy/go-control-plane/pkg/server/v3"
	service_discovery "github.com/upper-institute/ops-control/gen/api/service-discovery"
)

type EnvoyDiscoveryOptions struct {
	NodeID                 string
	GRPCServer             *grpc.Server
	Services               []EnvoyDiscoveryService
	WatchInterval          time.Duration
	ServiceDiscoverTimeout time.Duration
}

func (options *EnvoyDiscoveryOptions) NewServer(ctx context.Context, logger *zap.SugaredLogger) (*envoyDiscoveryServer, error) {

	cache := cache.NewSnapshotCache(false, cache.IDHash{}, nil)

	server := serverv3.NewServer(ctx, cache, nil)

	discoverygrpc.RegisterAggregatedDiscoveryServiceServer(options.GRPCServer, server)
	endpointservice.RegisterEndpointDiscoveryServiceServer(options.GRPCServer, server)
	clusterservice.RegisterClusterDiscoveryServiceServer(options.GRPCServer, server)
	routeservice.RegisterRouteDiscoveryServiceServer(options.GRPCServer, server)
	listenerservice.RegisterListenerDiscoveryServiceServer(options.GRPCServer, server)
	secretservice.RegisterSecretDiscoveryServiceServer(options.GRPCServer, server)
	runtimeservice.RegisterRuntimeDiscoveryServiceServer(options.GRPCServer, server)

	return &envoyDiscoveryServer{
		options: options,
		logger:  logger.With("part", "service-mesh/envoy-discovery-service"),
		server:  server,
		cache:   cache,
	}, nil

}

type envoyDiscoveryServer struct {
	options *EnvoyDiscoveryOptions

	logger *zap.SugaredLogger

	server serverv3.Server

	cache   cache.SnapshotCache
	version int64
}

func (e *envoyDiscoveryServer) discover() {

	version := int64(0)
	hash := []byte{}

	for {

		wg := sync.WaitGroup{}
		ctx, cancel := context.WithCancel(context.Background())

		if e.options.ServiceDiscoverTimeout > 0 {
			ctx, cancel = context.WithTimeout(ctx, e.options.ServiceDiscoverTimeout)
		}

		applySvcCh := make(chan *service_discovery.Service)
		res := NewResources()

		go func() {

			for applySvc := range applySvcCh {
				res.ApplyService(applySvc)
			}

		}()

		// Execute discovery services in parallel

		for _, s := range e.options.Services {

			wg.Add(1)

			go func(service EnvoyDiscoveryService) {

				defer wg.Done()

				svcCh := make(chan *service_discovery.Service)

				go service.Discover(ctx, svcCh)

				for applySvc := range svcCh {
					applySvcCh <- applySvc
				}

			}(s)

		}

		wg.Wait()

		close(applySvcCh)
		cancel()

		// Update cache if snapshot hash doesn't match

		newHash := res.Hash()

		if !bytes.Equal(hash, newHash) {

			snapshot, err := res.DoSnapshot(version)
			if err != nil {
				e.logger.Error(err)
				continue
			}

			err = e.cache.SetSnapshot(ctx, e.options.NodeID, snapshot)
			if err != nil {
				e.logger.Error(err)
				continue
			}

			hash = newHash

		}

		time.Sleep(e.options.WatchInterval)

	}

}

func (e *envoyDiscoveryServer) StartDiscoveryCycle() error {

	go e.discover()

	return nil

}
