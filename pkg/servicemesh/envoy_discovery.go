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
	sdapi "github.com/upper-institute/hike/proto/api/service-discovery"
)

type EnvoyDiscoveryOptions struct {
	NodeID                 string
	Services               []EnvoyDiscoveryService
	WatchInterval          time.Duration
	ServiceDiscoverTimeout time.Duration
}

func (options *EnvoyDiscoveryOptions) NewServer(ctx context.Context, logger *zap.SugaredLogger) (*EnvoyDiscoveryServer, error) {

	cache := cache.NewSnapshotCache(true, cache.IDHash{}, nil)

	server := serverv3.NewServer(ctx, cache, nil)

	return &EnvoyDiscoveryServer{
		options: options,
		logger:  logger.With("part", "service-mesh/envoy-discovery-service"),
		server:  server,
		cache:   cache,
	}, nil

}

type EnvoyDiscoveryServer struct {
	options *EnvoyDiscoveryOptions

	logger *zap.SugaredLogger

	server serverv3.Server

	cache   cache.SnapshotCache
	version int64
}

func (e *EnvoyDiscoveryServer) Register(grpcServer *grpc.Server) {

	discoverygrpc.RegisterAggregatedDiscoveryServiceServer(grpcServer, e.server)
	endpointservice.RegisterEndpointDiscoveryServiceServer(grpcServer, e.server)
	clusterservice.RegisterClusterDiscoveryServiceServer(grpcServer, e.server)
	routeservice.RegisterRouteDiscoveryServiceServer(grpcServer, e.server)
	listenerservice.RegisterListenerDiscoveryServiceServer(grpcServer, e.server)
	secretservice.RegisterSecretDiscoveryServiceServer(grpcServer, e.server)
	runtimeservice.RegisterRuntimeDiscoveryServiceServer(grpcServer, e.server)
	routeservice.RegisterVirtualHostDiscoveryServiceServer(grpcServer, e.server)

}

func (e *EnvoyDiscoveryServer) discover() {

	version := int64(0)
	hash := []byte{}

	for {

		e.logger.Info("New discover cycle")

		wg := sync.WaitGroup{}
		ctx, cancel := context.WithCancel(context.Background())

		if e.options.ServiceDiscoverTimeout > 0 {
			ctx, cancel = context.WithTimeout(ctx, e.options.ServiceDiscoverTimeout)
		}

		applySvcCh := make(chan *sdapi.Service)
		res := NewResources(e.logger)

		go func() {

			for applySvc := range applySvcCh {
				res.ApplyService(applySvc)
			}

		}()

		// Execute discovery services in parallel

		e.logger.Infow("Execute discovery services", "service_count", len(e.options.Services))

		for _, s := range e.options.Services {

			wg.Add(1)

			go func(service EnvoyDiscoveryService) {

				defer wg.Done()

				svcCh := make(chan *sdapi.Service)

				go service.Discover(ctx, svcCh)

				for applySvc := range svcCh {
					applySvcCh <- applySvc
				}

			}(s)

		}

		e.logger.Info("Waiting for discovery services...")

		wg.Wait()
		close(applySvcCh)

		// Update cache if snapshot hash doesn't match

		e.logger.Info("Updating resources cache")

		newHash := res.Hash()

		if !bytes.Equal(hash, newHash) {

			e.logger.Infow("Need snapshot", "node_id", e.options.NodeID)

			snapshot, err := res.DoSnapshot(version)
			if err != nil {
				e.logger.Error(err)
				cancel()
				continue
			}

			err = e.cache.SetSnapshot(ctx, e.options.NodeID, snapshot)
			if err != nil {
				e.logger.Error(err)
				cancel()
				continue
			}

			version++
			hash = newHash

		}

		cancel()

		e.logger.Infow("Sleeping before new discover cycle", "watch_interval", e.options.WatchInterval, "version", version)

		time.Sleep(e.options.WatchInterval)

	}

}

func (e *EnvoyDiscoveryServer) StartDiscoveryCycle() error {

	go e.discover()

	return nil

}
